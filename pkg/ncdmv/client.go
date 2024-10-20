package ncdmv

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/chromedp"
	"github.com/gtuk/discordwebhook"
	"golang.org/x/exp/slices"
	"golang.org/x/exp/slog"

	"github.com/aksiksi/ncdmv/pkg/models"
)

const (
	discordWebhookUsername = "ncdmv-bot"

	makeApptUrl = "https://skiptheline.ncdot.gov/"

	// Selectors
	makeApptButtonSelector               = "button#cmdMakeAppt"
	appointmentCalendarSelector          = "div.CalendarDateModel.hasDatepicker"
	appointmentDaySelector               = `td[data-handler="selectDay"]`
	appointmentDayLinkSelector           = `td[data-handler="selectDay"] > a.ui-state-default`
	appointmentTimeDropdownSelector      = "div.AppointmentTime select"
	loadingSpinnerSelector               = "div.blockUI"
	appointmentCalendarNextMonthSelector = "a.ui-datepicker-next"

	// Class and attribute names
	locationAvailableClassName       = "Active-Unit"
	appointmentMonthAttributeName    = "data-month"
	appointmentYearAttributeName     = "data-year"
	appointmentDatetimeAttributeName = "data-datetime"
	appointmentTypeIDAttributeName   = "data-appointmenttypeid"

	appointmentTimeFormat = "1/2/2006 3:04:05 PM"

	numAppointmentsPerDiscordNotification = 10

	temporaryErrString = "Could not find node with given id"

	defaultWaitTimeout = 30 * time.Second
)

var tz = loadTimezoneUnchecked("America/New_York")

func loadTimezoneUnchecked(tz string) *time.Location {
	loc, err := time.LoadLocation(tz)
	if err != nil {
		slog.Error("Failed to load timezone", "tz", tz, "err", err)
		os.Exit(1)
	}
	return loc
}

// isLocationNodeEnabled returns "true" if the location DOM node is available/clickable.
func isLocationNodeEnabled(node *cdp.Node) bool {
	return strings.Contains(node.AttributeValue("class"), locationAvailableClassName)
}

type Client struct {
	db                *models.Queries
	discordWebhook    string
	stopOnFailure     bool
	notifyUnavailable bool
}

func NewClient(db *sql.DB, discordWebhook string, stopOnFailure, notifyUnavailable bool) *Client {
	return &Client{
		db:                models.New(db),
		discordWebhook:    discordWebhook,
		stopOnFailure:     stopOnFailure,
		notifyUnavailable: notifyUnavailable,
	}
}

func isLocationAvailable(ctx context.Context, apptType AppointmentType, location Location) (bool, error) {
	// Wait for the location and read the node.
	var nodes []*cdp.Node
	if err := chromedp.Run(ctx,
		chromedp.WaitVisible(location.ToSelector(), chromedp.ByQuery),
		chromedp.Nodes(location.ToSelector(), &nodes, chromedp.NodeVisible, chromedp.ByQuery),
	); err != nil {
		return false, err
	}

	if len(nodes) == 0 {
		return false, fmt.Errorf("found no nodes for location %q - is it even valid?", location)
	} else if len(nodes) != 1 {
		return false, fmt.Errorf("found multiple nodes for location %q: %+v", location, nodes)
	}

	return isLocationNodeEnabled(nodes[0]), nil
}

type Appointment struct {
	Location Location
	Time     time.Time
}

func (a Appointment) String() string {
	return fmt.Sprintf("Appointment(location: %q, time: %s)", a.Location, a.Time)
}

// extractAppointmentTimesForDay lists all of the appointments available for the selected
// day in the calendar.
func extractAppointmentTimesForDay(ctx context.Context, apptType AppointmentType) ([]time.Time, error) {
	// This selects options from the appointment time dropdown that match the selected appointment type.
	optionSelector := fmt.Sprintf(`option[%s="%d"]`, appointmentTypeIDAttributeName, apptType)

	ctx, cancel := context.WithTimeout(ctx, defaultWaitTimeout)
	defer cancel()

	var timeDropdownHtml string
	if err := chromedp.Run(ctx,
		// Wait for the time dropdown element to contain valid appointment time options.
		chromedp.WaitReady(optionSelector, chromedp.ByQuery),

		// Extract the HTML for the time dropdown.
		chromedp.OuterHTML(appointmentTimeDropdownSelector, &timeDropdownHtml, chromedp.ByQuery),
	); err != nil {
		if ctx.Err() != nil {
			// No valid times were found in the dropdown.
			return nil, nil
		}
		return nil, err
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(timeDropdownHtml))
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	var availableTimes []time.Time
	doc.Find(optionSelector).Each(func(i int, s *goquery.Selection) {
		dt, ok := s.Attr(appointmentDatetimeAttributeName)
		if !ok {
			return
		}
		t, err := time.ParseInLocation(appointmentTimeFormat, dt, tz)
		if err != nil {
			slog.Error("Failed to parse datetime", "dt", dt, "err", err)
			return
		}
		availableTimes = append(availableTimes, t)
	})

	return availableTimes, nil
}

// findAvailableAppointmentDateNodeIDs finds all available dates on the location calendar page
// for the current/selected month and returns their node IDs.
func findAvailableAppointmentDateNodeIDs(ctx context.Context) ([]cdp.NodeID, error) {
	// NodeIDs will block until we find at least one matching node for the selector. But, in cases where
	// the calendar view has no clickable days, we still want to try to check the next month.
	//
	// To circumvent this behavior, we define a new context timeout that will be cancelled if no node is found.
	//
	// See: https://github.com/chromedp/chromedp/issues/379
	ctx, cancel := context.WithTimeout(ctx, defaultWaitTimeout)
	defer cancel()

	var nodeIDs []cdp.NodeID
	if err := chromedp.Run(ctx,
		// Wait for the spinner to disappear.
		chromedp.WaitNotPresent(loadingSpinnerSelector, chromedp.ByQuery),

		// Find all active/clickable day nodes.
		chromedp.NodeIDs(appointmentDayLinkSelector, &nodeIDs, chromedp.ByQueryAll),
	); err != nil {
		if ctx.Err() != nil {
			// If the context was cancelled, it just means that no clickable date nodes were found
			// on the page.
			return nil, nil
		}
		return nil, err
	}
	return nodeIDs, nil
}

// navigateAppointmentCalendarDays clicks each open day on the calendar for the current month
// and returns all available time slots.
func navigateAppointmentCalendarDays(ctx context.Context, apptType AppointmentType) (appointmentTimes []time.Time, _ error) {
	// Find the node IDs for the available dates.
	nodeIDs, err := findAvailableAppointmentDateNodeIDs(ctx)
	if err != nil {
		return nil, err
	}
	numNodes := len(nodeIDs)

	for i := 0; i < numNodes; i++ {
		nodeID := nodeIDs[i]

		if err := chromedp.Run(ctx,
			chromedp.Click([]cdp.NodeID{nodeID}, chromedp.ByNodeID),

			// Wait for the spinner to appear.
			chromedp.WaitReady(loadingSpinnerSelector, chromedp.ByQuery),

			// Wait for the spinner to disappear.
			chromedp.WaitNotPresent(loadingSpinnerSelector, chromedp.ByQuery),
		); err != nil {
			return nil, err
		}

		// Extract appointment times for the current date.
		times, err := extractAppointmentTimesForDay(ctx, apptType)
		if err != nil {
			return nil, err
		}

		// Refresh node IDs after clicking each date.
		nodeIDs, err = findAvailableAppointmentDateNodeIDs(ctx)
		if err != nil {
			return nil, err
		}
		if len(nodeIDs) != numNodes {
			// The calendar UI has changed. We can't proceed.
			return nil, fmt.Errorf("original node count (%d) != new node count (%d)", numNodes, len(nodeIDs))
		}

		appointmentTimes = append(appointmentTimes, times...)
	}

	return appointmentTimes, nil
}

// navigateAppointmentCalendar starts on the calendar page and finds all available appointments.
// It then keeps clicking on the right arrow and repeating the process for each month. It stops
// once the arrow becomes inactive (no more months).
func navigateAppointmentCalendar(ctx context.Context, apptType AppointmentType) (appointmentTimes []time.Time, _ error) {
	if err := chromedp.Run(ctx,
		// Wait for the spinner to appear.
		chromedp.WaitReady(loadingSpinnerSelector, chromedp.ByQuery),

		// Wait for the spinner to disappear.
		chromedp.WaitNotPresent(loadingSpinnerSelector, chromedp.ByQuery),
	); err != nil {
		return nil, err
	}

	for {
		// Click through all of the available days in the current month to find available
		// appointment times.
		times, err := navigateAppointmentCalendarDays(ctx, apptType)
		if err != nil {
			return nil, err
		}
		appointmentTimes = append(appointmentTimes, times...)

		// Figure out if the next month button is clickable.
		var attrValue string
		var attrExists bool
		var nodeIDs []cdp.NodeID
		if err := chromedp.Run(ctx,
			chromedp.AttributeValue(appointmentCalendarNextMonthSelector, "data-handler", &attrValue, &attrExists, chromedp.ByQuery),
			chromedp.NodeIDs(appointmentCalendarNextMonthSelector, &nodeIDs, chromedp.ByQuery),
		); err != nil {
			return nil, err
		}
		if !attrExists {
			break
		}

		// Click the next month button.
		if err := chromedp.Run(ctx, chromedp.Click([]cdp.NodeID{nodeIDs[0]}, chromedp.ByNodeID)); err != nil {
			return nil, err
		}
	}

	return appointmentTimes, nil
}

// appointmentFlowState represents the current state of the appointment workflow for a single location.
type appointmentFlowState int

const (
	appointmentFlowStateStart appointmentFlowState = iota
	appointmentFlowStateMainPage
	appointmentFlowStateAppointmentType
	appointmentFlowStateLocationsPage
	appointmentFlowStateLocationCalendar
)

// findAvailableAppointments finds all available appointment dates for the given location.
//
// This function uses a simple state machine to navigate the appointment flow.
//
// NOTE: Currently does not parse the appointment time slots - just dates. Also, this does not look at
// later months.
func findAvailableAppointments(ctx context.Context, apptType AppointmentType, location Location) (appointments []*Appointment, _ error) {
	state := appointmentFlowStateStart

	for {
		switch state {
		case appointmentFlowStateStart:
			// Navigate to the main page.
			if _, err := chromedp.RunResponse(ctx, chromedp.Navigate(makeApptUrl)); err != nil {
				return nil, err
			}
			state = appointmentFlowStateMainPage
		case appointmentFlowStateMainPage:
			// Click the "Make Appointment" button once it is visible.
			if _, err := chromedp.RunResponse(ctx, chromedp.Click(makeApptButtonSelector, chromedp.NodeVisible, chromedp.ByQuery)); err != nil {
				return nil, err
			}
			state = appointmentFlowStateAppointmentType
		case appointmentFlowStateAppointmentType:
			// Click the appointment type button.
			if _, err := chromedp.RunResponse(ctx, chromedp.Click(apptType.ToSelector(), chromedp.NodeVisible, chromedp.ByQuery)); err != nil {
				return nil, err
			}
			state = appointmentFlowStateLocationsPage
		case appointmentFlowStateLocationsPage:
			// Check if the location is available.
			isAvailable, err := isLocationAvailable(ctx, apptType, location)
			if err != nil {
				return nil, err
			}
			// If it isn't, it means no appointments are available.
			if !isAvailable {
				return nil, nil
			}
			// At this point, we are on the locations page. Click the location button.
			if _, err := chromedp.RunResponse(ctx, chromedp.Click(location.ToSelector())); err != nil {
				return nil, err
			}
			state = appointmentFlowStateLocationCalendar
		case appointmentFlowStateLocationCalendar:
			// Find available dates for this location by parsing the calendar HTML.
			appointmentTimes, err := navigateAppointmentCalendar(ctx, apptType)
			if err != nil {
				return nil, err
			}
			for _, d := range appointmentTimes {
				appointments = append(appointments, &Appointment{
					Location: location,
					Time:     d,
				})
			}
			return appointments, nil
		}
	}
}

func (c Client) sendDiscordMessage(content string) error {
	if c.discordWebhook == "" {
		// Nothing to do here.
		return nil
	}

	username := discordWebhookUsername
	if err := discordwebhook.SendMessage(c.discordWebhook, discordwebhook.Message{
		Username: &username,
		Content:  &content,
	}); err != nil {
		return fmt.Errorf("failed to send message to Discord webhook: %w", err)
	}

	return nil
}

// RunForLocations finds all available appointments across the given locations.
//
// NOTE: For now, this only looks at _appointment dates_ and only considers the first available month.
func (c Client) RunForLocations(ctx context.Context, apptType AppointmentType, locations []Location, timeout time.Duration) ([]*Appointment, error) {
	// Common timeout for all locations.
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Setup a seperate tab context for each location. The tabs will be closed when this function
	// returns.
	var locationCtxs []context.Context
	var locationCtxCancels []context.CancelFunc
	for range locations {
		ctx, cancel := chromedp.NewContext(ctx)
		defer cancel()
		locationCtxs = append(locationCtxs, ctx)
		locationCtxCancels = append(locationCtxCancels, cancel)
	}

	type locationResult struct {
		idx          int
		appointments []*Appointment
		err          error
	}
	resultChan := make(chan locationResult)

	// Spawn a goroutine for each location. Each location is processed in a separate
	// browser tab. Once processing completes for a location, its tab will be closed.
	for i, location := range locations {
		i, location := i, location
		ctx, cancel := locationCtxs[i], locationCtxCancels[i]
		go func() {
			slog.Debug("Starting to process location...", "location", location)
			appointments, err := findAvailableAppointments(ctx, apptType, location)
			resultChan <- locationResult{
				idx:          i,
				appointments: appointments,
				err:          err,
			}
			// Cancelling the context closes the tab for the given location.
			cancel()
		}()
	}

	// Extract appointments from all of the locations.
	var appointments []*Appointment
	for i := 0; i < len(locations); i++ {
		result := <-resultChan
		location := locations[result.idx]

		if result.err != nil {
			return nil, result.err
		}
		if len(result.appointments) == 0 {
			slog.Info("No appointments available", "location", location)
		} else {
			slog.Info("Found appointments in location", "location", location, "num_appointments", len(result.appointments))
		}
		appointments = append(appointments, result.appointments...)
	}

	return appointments, nil
}

func (c Client) sendNotifications(ctx context.Context, apptType AppointmentType, appointmentsToNotify []models.Appointment, discordWebhook string, interval time.Duration) error {
	// Sort appointments by time.
	slices.SortFunc(appointmentsToNotify, func(a, b models.Appointment) int {
		if a.Time.Before(b.Time) {
			return -1
		} else if a.Time.After(b.Time) {
			return 1
		}
		return 0
	})

	// Group appointments by location.
	appointmentsByLocation := make(map[string][]models.Appointment)
	for _, a := range appointmentsToNotify {
		appointmentsByLocation[a.Location] = append(appointmentsByLocation[a.Location], a)
	}

	// Sort locations by name.
	var locations []string
	for location := range appointmentsByLocation {
		locations = append(locations, location)
	}
	slices.Sort(locations)

	for i, location := range locations {
		appointments := appointmentsByLocation[location]
		b := strings.Builder{}

		// If this is the first location, start with a header. Later messages will just be
		// a continuation of the first one.
		if i == 0 {
			if c.notifyUnavailable {
				b.WriteString("Found appointment change(s) at the following locations and times:\n")
			} else {
				b.WriteString("Found available appointment(s) at the following locations and times:\n")
			}
		}

		b.WriteString(fmt.Sprintf("\n- **%s**:\n", location))

		// Construct a list bullet for each appointment change for this location.
		for i, appointment := range appointments {
			if i == numAppointmentsPerDiscordNotification {
				b.WriteString("  - `(... more appointments available)`\n")
				break
			}
			if appointment.Available {
				b.WriteString(fmt.Sprintf("  - :white_check_mark: `%s`\n", appointment.Time.String()))
			} else if c.notifyUnavailable {
				b.WriteString(fmt.Sprintf("  - :x: `%s`\n", appointment.Time.String()))
			}
		}

		if i == len(locations)-1 {
			// The last message includes a link to the NCDMV appointment page.
			b.WriteString("\nBook an appointment here: https://skiptheline.ncdot.gov")
		}

		// Send the Discord message.
		if err := c.sendDiscordMessage(b.String()); err != nil {
			log.Printf("Failed to send message to Discord webhook %q: %v", c.discordWebhook, err)
			continue
		}

		// Mark all of the appointments in the batch as "notified".
		for _, appointment := range appointments {
			if _, err := c.db.CreateNotification(ctx, models.CreateNotificationParams{
				AppointmentID:  appointment.ID,
				DiscordWebhook: sql.NullString{String: discordWebhook, Valid: true},
				Available:      appointment.Available,
				ApptType:       apptType.String(),
			}); err != nil {
				return fmt.Errorf("failed to create notification for appointment %v: %w", appointment, err)
			}
		}

		time.Sleep(interval)
	}

	return nil
}

func findAppointmentsToUpdateAndNotify(new, existing []models.Appointment, locations []Location) (toUpdate, toNotify []models.Appointment) {
	newAppointments := make(map[ /* ID */ int64]models.Appointment)
	existingAppointments := make(map[ /* ID */ int64]models.Appointment)
	for _, a := range new {
		newAppointments[a.ID] = a
	}
	for _, a := range existing {
		existingAppointments[a.ID] = a
	}

	// Find appointments that are either new or were previously unavailable.
	for id, appt := range newAppointments {
		// Notify on new appointments.
		existingAppt, ok := existingAppointments[id]
		if !ok {
			toNotify = append(toNotify, appt)
			continue
		}

		// Notify and update appointments with an availability change.
		if appt.Available != existingAppt.Available {
			toNotify = append(toNotify, appt)
			toUpdate = append(toUpdate, appt)
		}
	}

	// Find appointments that are now (implicitly) unavailable for locations that the user is
	// interested in.
	for id, appt := range existingAppointments {
		isLocationTracked := slices.ContainsFunc(locations, func(l Location) bool {
			return appt.Location == l.String()
		})
		if !isLocationTracked {
			continue
		}
		_, ok := newAppointments[id]
		if !ok && appt.Available {
			appt.Available = false
			toNotify = append(toNotify, appt)
			toUpdate = append(toUpdate, appt)
		}
	}

	return toUpdate, toNotify
}

func (c Client) updateAppointments(ctx context.Context, appointmentsToUpdate []models.Appointment) error {
	for _, appt := range appointmentsToUpdate {
		if err := c.db.UpdateAppointmentAvailable(ctx, models.UpdateAppointmentAvailableParams{
			ID:        appt.ID,
			Available: appt.Available,
		}); err != nil {
			return fmt.Errorf("failed to update appointment %d: %w", appt.ID, err)
		}
	}
	return nil
}

// listExistingAppointmentsInLocations lists all existing appointments after the provided date for the given locations.
func (c Client) listExistingAppointmentsInLocations(ctx context.Context, t time.Time, locations []Location) ([]models.Appointment, error) {
	var locationStrings []string
	for _, loc := range locations {
		locationStrings = append(locationStrings, loc.String())
	}
	existingAppointments, err := c.db.ListAppointmentsAfterDateForLocations(ctx, models.ListAppointmentsAfterDateForLocationsParams{
		Time:      t,
		Locations: locationStrings,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list appointments after current time (%v) for locations %v: %w", t, locations, err)
	}
	return existingAppointments, nil
}

func (c Client) handleTick(ctx context.Context, apptType AppointmentType, locations []Location, timeout time.Duration) error {
	now := time.Now()

	// Prune all invalid appointments (i.e., those that are in the past) by setting them as unavailable.
	rows, err := c.db.PruneAppointmentsBeforeDate(ctx, now)
	if err != nil {
		return fmt.Errorf("failed to delete appointments before current time (%v): %w", now, err)
	}
	if len(rows) > 0 {
		slog.Info("Pruned invalid appointments", "count", len(rows))
	}

	existingAppointments, err := c.listExistingAppointmentsInLocations(ctx, now, locations)
	if err != nil {
		return err
	}
	slog.Info("Listed existing appointments in provided locations", "count", len(existingAppointments))

	appointments, err := c.RunForLocations(ctx, apptType, locations, timeout)
	if err != nil {
		return fmt.Errorf("failed to check locations: %w", err)
	}

	// TODO(aksiksi): How do we handle failures after writing appointments to DB but before writing notifications?
	//
	// One idea is to keep track of the last update time for an appointment and check if a notification exists
	// that was sent _after_ that time.
	//
	// Currently, if this happens, we'd end up never notifying as the appointments were written to the DB.
	var newAppointments []models.Appointment
	for _, appointment := range appointments {
		exists := false
		a, err := c.db.CreateAppointment(ctx, models.CreateAppointmentParams{
			Location:  appointment.Location.String(),
			Time:      appointment.Time,
			Available: true,
		})
		if err != nil {
			slog.Debug("Appointment already processed", "location", appointment.Location.String(), "time", appointment.Time)
			exists = true
		}
		if exists {
			// Fetch the appointment ID from the DB.
			a, err = c.db.GetAppointmentByLocationAndTime(ctx, models.GetAppointmentByLocationAndTimeParams{
				Location: appointment.Location.String(),
				Time:     appointment.Time,
			})
			if err != nil {
				return fmt.Errorf("appointment %q does not exist in DB: %w", appointment, err)
			}
			a.Available = true
		}
		newAppointments = append(newAppointments, a)
	}

	appointmentsToUpdate, appointmentsToNotify := findAppointmentsToUpdateAndNotify(newAppointments, existingAppointments, locations)
	slog.Info("Found appointments to update and notify", "to_update", len(appointmentsToUpdate), "to_notify", len(appointmentsToNotify))

	if err := c.updateAppointments(ctx, appointmentsToUpdate); err != nil {
		return fmt.Errorf("failed to update existing appointments: %w", err)
	}
	if len(appointmentsToUpdate) > 0 {
		slog.Info("Updated appointments successfully", "count", len(appointmentsToUpdate))
	}

	if err := c.sendNotifications(ctx, apptType, appointmentsToNotify, c.discordWebhook, 1*time.Second); err != nil {
		return fmt.Errorf("failed to send notifications: %w", err)
	}
	if len(appointmentsToNotify) > 0 {
		slog.Info("Sent notifications successfully", "count", len(appointmentsToNotify))
	}

	return nil
}

// Start runs the NC DMV client for the given locations. A search will be run for all locations based on
// the specified interval.
//
// Note that this method will block until the context is cancelled. If you want to just run a single search synchronously,
// you should use RunForLocations.
//
// If stopOnFailure is set to true, this method will terminate on the first error.
//
// Each provided location is processed in a _separate_ Chrome browser tab. This allows for some degree of parallelism
// as each tab can run independently of the others. The downside is that the list of locations needs to bounded based
// on the resources available on your machine.
func (c Client) Start(ctx context.Context, apptType AppointmentType, locations []Location, timeout, interval time.Duration) error {
	t := time.NewTicker(interval)
	defer t.Stop()

	slog.Info("Starting client", "appt_type", apptType, "locations", locations, "timeout", timeout, "interval", interval)

	tick := func() error {
		defer slog.Info("Sleeping between location checks...", "interval", interval)
		for {
			if err := c.handleTick(ctx, apptType, locations, timeout); err != nil {
				if strings.Contains(err.Error(), temporaryErrString) {
					slog.Warn("handleTick failed with temporary error; retrying tick...")
					continue
				}
				slog.Error("handleTick failed", "err", err)
				if c.stopOnFailure {
					return err
				}
			}
			return nil
		}
	}

	for {
		// Trigger a "tick" immediately as the ticker does not do so for us.
		if err := tick(); err != nil {
			return err
		}
		// Block until the next tick or the context is cancelled.
		select {
		case <-t.C:
			if err := tick(); err != nil {
				return err
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}
