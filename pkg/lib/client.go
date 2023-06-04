package lib

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/chromedp"
	"github.com/gtuk/discordwebhook"
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
)

var tz = loadTimezoneUnchecked("America/New_York")

func loadTimezoneUnchecked(tz string) *time.Location {
	loc, err := time.LoadLocation(tz)
	if err != nil {
		log.Fatalf("Failed to load timezone %q: %v", tz, err)
	}
	return loc
}

// isLocationNodeEnabled returns "true" if the location DOM node is available/clickable.
func isLocationNodeEnabled(node *cdp.Node) bool {
	return strings.Contains(node.AttributeValue("class"), locationAvailableClassName)
}

type Client struct {
	// chromedp browser context.
	ctx                      context.Context
	discordWebhook           string
	stopOnFailure            bool
	appointmentNotifications map[Appointment]bool
}

func NewClient(ctx context.Context, discordWebhook string, headless, disableGpu, debug, stopOnFailure bool) (*Client, context.CancelFunc, error) {
	allocatorOpts := chromedp.DefaultExecAllocatorOptions[:]
	var ctxOpts []chromedp.ContextOption
	if !headless {
		allocatorOpts = append(allocatorOpts, chromedp.Flag("headless", false))
	}
	if disableGpu {
		allocatorOpts = append(allocatorOpts, chromedp.DisableGPU)
	}
	if debug {
		ctxOpts = append(ctxOpts, chromedp.WithDebugf(log.Printf))
	}

	ctx, cancel1 := chromedp.NewExecAllocator(ctx, allocatorOpts...)
	ctx, cancel2 := chromedp.NewContext(ctx, ctxOpts...)
	cancel := func() { cancel2(); cancel1() }

	// Open the first (empty) tab.
	if err := chromedp.Run(ctx); err != nil {
		cancel()
		return nil, nil, fmt.Errorf("failed to open first tab: %w", err)
	}

	return &Client{
		ctx:                      ctx,
		discordWebhook:           discordWebhook,
		stopOnFailure:            stopOnFailure,
		appointmentNotifications: make(map[Appointment]bool),
	}, cancel, nil
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

	var timeDropdownHtml string
	if err := chromedp.Run(ctx,
		// Wait for the time dropdown element to contain valid appointment time options.
		chromedp.WaitReady(optionSelector, chromedp.ByQuery),

		// Extract the HTML for the time dropdown.
		chromedp.OuterHTML(appointmentTimeDropdownSelector, &timeDropdownHtml, chromedp.ByQuery),
	); err != nil {
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
			log.Printf("Failed to parse datetime %q: %v", dt, err)
			return
		}
		availableTimes = append(availableTimes, t)
	})

	return availableTimes, nil
}

// findAvailableAppointmentDateNodeIDs finds all available dates on the location calendar page
// for the current/selected month and returns their node IDs.
func findAvailableAppointmentDateNodeIDs(ctx context.Context) ([]cdp.NodeID, error) {
	var nodeIDs []cdp.NodeID
	if err := chromedp.Run(ctx,
		// Wait for the spinner to disappear.
		chromedp.WaitNotPresent(loadingSpinnerSelector, chromedp.ByQuery),

		// Find all active/clickable day nodes.
		chromedp.NodeIDs(appointmentDayLinkSelector, &nodeIDs, chromedp.NodeEnabled, chromedp.ByQueryAll),
	); err != nil {
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

func (c Client) sendDiscordMessage(appointment Appointment) error {
	if c.discordWebhook == "" {
		// Nothing to do here.
		return nil
	}

	// Send a notification for this appointment if we haven't already done so.
	if !c.appointmentNotifications[appointment] {
		username := discordWebhookUsername
		content := fmt.Sprintf("Found appointment: %q, Book here: https://skiptheline.ncdot.gov", appointment)
		if err := discordwebhook.SendMessage(c.discordWebhook, discordwebhook.Message{
			Username: &username,
			Content:  &content,
		}); err != nil {
			return fmt.Errorf("failed to send message to Discord webhook: %w", err)
		}
		c.appointmentNotifications[appointment] = true
	}

	return nil
}

// RunForLocations finds all available appointments across the given locations.
//
// NOTE: For now, this only looks at _appointment dates_ and only considers the first available month.
func (c Client) RunForLocations(apptType AppointmentType, locations []Location, timeout time.Duration) ([]*Appointment, error) {
	// Common timeout for all locations.
	ctx, cancel := context.WithTimeout(c.ctx, timeout)
	defer cancel()

	// Setup a seperate tab context for each location. The tabs will be closed when this function
	// returns.
	var locationCtxs []context.Context
	for range locations {
		ctx, cancel := chromedp.NewContext(ctx)
		defer cancel()
		locationCtxs = append(locationCtxs, ctx)
	}

	type locationResult struct {
		idx          int
		appointments []*Appointment
		err          error
	}
	resultChan := make(chan locationResult)

	// Spawn a goroutine for each location. Each locations is processed in a separate
	// browser tab.
	for i, location := range locations {
		i := i
		location := location
		ctx := locationCtxs[i]
		go func() {
			appointments, err := findAvailableAppointments(ctx, apptType, location)
			resultChan <- locationResult{
				idx:          i,
				appointments: appointments,
				err:          err,
			}
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
			log.Printf("Location %q has no appointments available", location)
		}
		appointments = append(appointments, result.appointments...)
	}

	return appointments, nil
}

// Start runs the NC DMV client for the given locations. A search will be run for all locations based on
// the specified interval.
//
// Note that this method will block indefinitely. If you want to just run a single search, use RunForLocations.
//
// If "stopOnFailure" is true for this client, this method will return any error encountered.
func (c Client) Start(apptType AppointmentType, locations []Location, timeout, interval time.Duration) error {
	for {
		appointments, err := c.RunForLocations(apptType, locations, timeout)
		if err != nil {
			if c.stopOnFailure {
				return fmt.Errorf("failed to check locations: %w", err)
			} else {
				log.Printf("Failed to check locations: %v", err)
			}
		}

		for _, appointment := range appointments {
			log.Printf("Found appointment: %q", appointment)
			if err := c.sendDiscordMessage(*appointment); err != nil {
				log.Printf("Failed to send message to Discord webhook %q: %v", c.discordWebhook, err)
			}
		}

		log.Printf("Sleeping for %v between checks...", interval)
		time.Sleep(interval)
	}
}
