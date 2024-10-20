package ncdmv

import (
	"fmt"
)

const (
	apptTypeSelector = `div.QflowObjectItem[data-id="%d"]`
	locationSelector = apptTypeSelector
)

func mapToKeys[K comparable, V any](m map[K]V) []string {
	var keys []string
	for k := range m {
		keys = append(keys, fmt.Sprintf("%v", k))
	}
	return keys
}

// AppointmentType represents the type of appointment.
// The value is the index of the box in the UI (see: "data-id").
type AppointmentType int

func (a AppointmentType) ToSelector() string {
	return fmt.Sprintf(apptTypeSelector, a)
}

const (
	AppointmentTypeInvalid                AppointmentType = iota
	AppointmentTypeDriverLicense          AppointmentType = 1
	AppointmentTypeDriverLicenseDuplicate AppointmentType = 2
	AppointmentTypeDriverLicenseRenewal   AppointmentType = 3
	AppointmentTypeIdCard                 AppointmentType = 5
	AppointmentTypeKnowledgeTest          AppointmentType = 6
	AppointmentTypeMotorcycleSkillsTest   AppointmentType = 8
	AppointmentTypeNonCDLRoadTest         AppointmentType = 13
	AppointmentTypePermit                 AppointmentType = 9
)

func (a AppointmentType) String() string {
	switch a {
	case AppointmentTypeInvalid:
		return "invalid"
	case AppointmentTypeDriverLicense:
		return "driver-license"
	case AppointmentTypeDriverLicenseDuplicate:
		return "driver-license-duplicate"
	case AppointmentTypeDriverLicenseRenewal:
		return "driver-license-renewal"
	case AppointmentTypeIdCard:
		return "id-card"
	case AppointmentTypeKnowledgeTest:
		return "knowledge-test"
	case AppointmentTypeMotorcycleSkillsTest:
		return "motorcycle-skills-test"
	case AppointmentTypeNonCDLRoadTest:
		return "non-cdl-road-test"
	case AppointmentTypePermit:
		return "permit"
	}
	panic("unreachable: invalid AppointmentType")
}

var appointmentTypeMap map[string]AppointmentType = map[string]AppointmentType{
	AppointmentTypeDriverLicense.String():          AppointmentTypeDriverLicense,
	AppointmentTypeDriverLicenseDuplicate.String(): AppointmentTypeDriverLicenseDuplicate,
	AppointmentTypeDriverLicenseRenewal.String():   AppointmentTypeDriverLicenseRenewal,
	AppointmentTypeIdCard.String():                 AppointmentTypeIdCard,
	AppointmentTypeKnowledgeTest.String():          AppointmentTypeKnowledgeTest,
	AppointmentTypeMotorcycleSkillsTest.String():   AppointmentTypeMotorcycleSkillsTest,
	AppointmentTypeNonCDLRoadTest.String():         AppointmentTypeNonCDLRoadTest,
	AppointmentTypePermit.String():                 AppointmentTypePermit,
}

func StringToAppointmentType(k string) AppointmentType {
	if v, ok := appointmentTypeMap[k]; !ok {
		return AppointmentTypeInvalid
	} else {
		return v
	}
}

func ValidApptTypes() []string {
	return mapToKeys(appointmentTypeMap)
}

type Location int

const (
	// Values taken from the "data-id" attribute set on each location div.
	LocationInvalid           Location = iota
	LocationAberdeen          Location = 100
	LocationAhoskie           Location = 99
	LocationAlbemarle         Location = 87
	LocationAndrews           Location = 142
	LocationAsheboro          Location = 97
	LocationAsheville         Location = 124
	LocationBoone             Location = 125
	LocationBrevard           Location = 101
	LocationBrysonCity        Location = 126
	LocationBurgaw            Location = 111
	LocationBurnsville        Location = 68
	LocationCarrboro          Location = 140
	LocationCary              Location = 66
	LocationCharlotteEast     Location = 120
	LocationCharlotteNorth    Location = 175
	LocationCharlotteSouth    Location = 86
	LocationCharlotteWest     Location = 121
	LocationClayton           Location = 42
	LocationClinton           Location = 112
	LocationClyde             Location = 102
	LocationConcord           Location = 141
	LocationDurhamEast        Location = 47
	LocationDurhamSouth       Location = 80
	LocationElizabethCity     Location = 65
	LocationElizabethtown     Location = 79
	LocationElkin             Location = 103
	LocationErwin             Location = 95
	LocationFayettevilleSouth Location = 118
	LocationFayettevilleWest  Location = 119
	LocationForestCity        Location = 57
	LocationFranklin          Location = 139
	LocationFuquayVarina      Location = 38
	LocationGarner            Location = 69
	LocationGastonia          Location = 59
	LocationGoldsboro         Location = 40
	LocationGraham            Location = 137
	LocationGreensboroEast    Location = 138
	LocationGreensboroWest    Location = 143
	LocationGreenville        Location = 22
	LocationHamlet            Location = 104
	LocationHavelock          Location = 82
	LocationHenderson         Location = 70
	LocationHendersonville    Location = 130
	LocationHickory           Location = 78
	LocationHighPoint         Location = 129
	LocationHillsborough      Location = 52
	LocationHudson            Location = 131
	LocationHuntersville      Location = 19
	LocationJacksonville      Location = 134
	LocationJefferson         Location = 83
	LocationKernersville      Location = 135
	LocationKinston           Location = 50
	LocationLexington         Location = 73
	LocationLincolnton        Location = 72
	LocationLouisburg         Location = 71
	LocationLumberton         Location = 48
	LocationMarion            Location = 62
	LocationMarshall          Location = 105
	LocationMocksville        Location = 91
	LocationMonroe            Location = 96
	LocationMooresville       Location = 110
	LocationMoreheadCity      Location = 41
	LocationMorganton         Location = 136
	LocationMountAiry         Location = 106
	LocationMountHolly        Location = 81
	LocationNagsHead          Location = 155
	LocationNewBern           Location = 43
	LocationNewton            Location = 107
	LocationOxford            Location = 64
	LocationPolkton           Location = 63
	LocationRaeford           Location = 115
	LocationRaleighEast       Location = 181
	LocationRaleighNorth      Location = 10
	LocationRaleighWest       Location = 9
	LocationRoanokeRapids     Location = 61
	LocationRoxboro           Location = 76
	LocationSalisbury         Location = 93
	LocationSanford           Location = 54
	LocationShallotte         Location = 113
	LocationShelby            Location = 58
	LocationSilerCity         Location = 109
	LocationSmithfield        Location = 44
	LocationStatesville       Location = 55
	LocationStedman           Location = 18
	LocationSylva             Location = 114
	LocationTarboro           Location = 60
	LocationTaylorsville      Location = 85
	LocationThomasville       Location = 94
	LocationTroy              Location = 77
	LocationWashington        Location = 89
	LocationWendell           Location = 39
	LocationWentworth         Location = 56
	LocationWhiteville        Location = 53
	LocationWilkesboro        Location = 116
	LocationWilliamston       Location = 88
	LocationWilmingtonNorth   Location = 123
	LocationWilmingtonSouth   Location = 132
	LocationWilson            Location = 45
	LocationWinstonSalemNorth Location = 51
	LocationWinstonSalemSouth Location = 122
	LocationYadkinville       Location = 128
)

func (l Location) ToSelector() string {
	return fmt.Sprintf(locationSelector, l)
}

func (l Location) String() string {
	switch l {
	case LocationAberdeen:
		return "aberdeen"
	case LocationAhoskie:
		return "ahoskie"
	case LocationAlbemarle:
		return "albemarle"
	case LocationAndrews:
		return "andrews"
	case LocationAsheboro:
		return "asheboro"
	case LocationAsheville:
		return "asheville"
	case LocationBoone:
		return "boone"
	case LocationBrevard:
		return "brevard"
	case LocationBrysonCity:
		return "bryson-city"
	case LocationBurgaw:
		return "burgaw"
	case LocationBurnsville:
		return "burnsville"
	case LocationCarrboro:
		return "carrboro"
	case LocationCary:
		return "cary"
	case LocationCharlotteEast:
		return "charlotte-east"
	case LocationCharlotteNorth:
		return "charlotte-north"
	case LocationCharlotteSouth:
		return "charlotte-south"
	case LocationCharlotteWest:
		return "charlotte-west"
	case LocationClayton:
		return "clayton"
	case LocationClinton:
		return "clinton"
	case LocationClyde:
		return "clyde"
	case LocationConcord:
		return "concord"
	case LocationDurhamEast:
		return "durham-east"
	case LocationDurhamSouth:
		return "durham-south"
	case LocationElizabethCity:
		return "elizabeth-city"
	case LocationElizabethtown:
		return "elizabethtown"
	case LocationElkin:
		return "elkin"
	case LocationErwin:
		return "erwin"
	case LocationFayettevilleSouth:
		return "fayetteville-south"
	case LocationFayettevilleWest:
		return "fayetteville-west"
	case LocationForestCity:
		return "forest-city"
	case LocationFranklin:
		return "franklin"
	case LocationFuquayVarina:
		return "fuquay-varina"
	case LocationGarner:
		return "garner"
	case LocationGastonia:
		return "gastonia"
	case LocationGoldsboro:
		return "goldsboro"
	case LocationGraham:
		return "graham"
	case LocationGreensboroEast:
		return "greensboro-east"
	case LocationGreensboroWest:
		return "greensboro-west"
	case LocationGreenville:
		return "greenville"
	case LocationHamlet:
		return "hamlet"
	case LocationHavelock:
		return "havelock"
	case LocationHenderson:
		return "henderson"
	case LocationHendersonville:
		return "hendersonville"
	case LocationHickory:
		return "hickory"
	case LocationHighPoint:
		return "high-point"
	case LocationHillsborough:
		return "hillsborough"
	case LocationHudson:
		return "hudson"
	case LocationHuntersville:
		return "huntersville"
	case LocationJacksonville:
		return "jacksonville"
	case LocationJefferson:
		return "jefferson"
	case LocationKernersville:
		return "kernersville"
	case LocationKinston:
		return "kinston"
	case LocationLexington:
		return "lexington"
	case LocationLincolnton:
		return "lincolnton"
	case LocationLouisburg:
		return "louisburg"
	case LocationLumberton:
		return "lumberton"
	case LocationMarion:
		return "marion"
	case LocationMarshall:
		return "marshall"
	case LocationMocksville:
		return "mocksville"
	case LocationMonroe:
		return "monroe"
	case LocationMooresville:
		return "mooresville"
	case LocationMoreheadCity:
		return "morehead-city"
	case LocationMorganton:
		return "morganton"
	case LocationMountAiry:
		return "mount-airy"
	case LocationMountHolly:
		return "mount-holly"
	case LocationNagsHead:
		return "nags-head"
	case LocationNewBern:
		return "new-bern"
	case LocationNewton:
		return "newton"
	case LocationOxford:
		return "oxford"
	case LocationPolkton:
		return "polkton"
	case LocationRaeford:
		return "raeford"
	case LocationRaleighEast:
		return "raleigh-east"
	case LocationRaleighNorth:
		return "raleigh-north"
	case LocationRaleighWest:
		return "raleigh-west"
	case LocationRoanokeRapids:
		return "roanoke-rapids"
	case LocationRoxboro:
		return "roxboro"
	case LocationSalisbury:
		return "salisbury"
	case LocationSanford:
		return "sanford"
	case LocationShallotte:
		return "shallotte"
	case LocationShelby:
		return "shelby"
	case LocationSilerCity:
		return "siler-city"
	case LocationSmithfield:
		return "smithfield"
	case LocationStatesville:
		return "statesville"
	case LocationStedman:
		return "stedman"
	case LocationSylva:
		return "sylva"
	case LocationTarboro:
		return "tarboro"
	case LocationTaylorsville:
		return "taylorsville"
	case LocationThomasville:
		return "thomasville"
	case LocationTroy:
		return "troy"
	case LocationWashington:
		return "washington"
	case LocationWendell:
		return "wendell"
	case LocationWentworth:
		return "wentworth"
	case LocationWhiteville:
		return "whiteville"
	case LocationWilkesboro:
		return "wilkesboro"
	case LocationWilliamston:
		return "williamston"
	case LocationWilmingtonNorth:
		return "wilmington-north"
	case LocationWilmingtonSouth:
		return "wilmington-south"
	case LocationWilson:
		return "wilson"
	case LocationWinstonSalemNorth:
		return "winstonsalem-north"
	case LocationWinstonSalemSouth:
		return "winstonsalem-south"
	case LocationYadkinville:
		return "yadkinville"
	}
	panic("unreachable: invalid Location")
}

var locationMap map[string]Location = map[string]Location{
	LocationAberdeen.String():          LocationAberdeen,
	LocationAhoskie.String():           LocationAhoskie,
	LocationAlbemarle.String():         LocationAlbemarle,
	LocationAndrews.String():           LocationAndrews,
	LocationAsheboro.String():          LocationAsheboro,
	LocationAsheville.String():         LocationAsheville,
	LocationBoone.String():             LocationBoone,
	LocationBrevard.String():           LocationBrevard,
	LocationBrysonCity.String():        LocationBrysonCity,
	LocationBurgaw.String():            LocationBurgaw,
	LocationBurnsville.String():        LocationBurnsville,
	LocationCarrboro.String():          LocationCarrboro,
	LocationCary.String():              LocationCary,
	LocationCharlotteEast.String():     LocationCharlotteEast,
	LocationCharlotteNorth.String():    LocationCharlotteNorth,
	LocationCharlotteSouth.String():    LocationCharlotteSouth,
	LocationCharlotteWest.String():     LocationCharlotteWest,
	LocationClayton.String():           LocationClayton,
	LocationClinton.String():           LocationClinton,
	LocationClyde.String():             LocationClyde,
	LocationConcord.String():           LocationConcord,
	LocationDurhamEast.String():        LocationDurhamEast,
	LocationDurhamSouth.String():       LocationDurhamSouth,
	LocationElizabethCity.String():     LocationElizabethCity,
	LocationElizabethtown.String():     LocationElizabethtown,
	LocationElkin.String():             LocationElkin,
	LocationErwin.String():             LocationErwin,
	LocationFayettevilleSouth.String(): LocationFayettevilleSouth,
	LocationFayettevilleWest.String():  LocationFayettevilleWest,
	LocationForestCity.String():        LocationForestCity,
	LocationFranklin.String():          LocationFranklin,
	LocationFuquayVarina.String():      LocationFuquayVarina,
	LocationGarner.String():            LocationGarner,
	LocationGastonia.String():          LocationGastonia,
	LocationGoldsboro.String():         LocationGoldsboro,
	LocationGraham.String():            LocationGraham,
	LocationGreensboroEast.String():    LocationGreensboroEast,
	LocationGreensboroWest.String():    LocationGreensboroWest,
	LocationGreenville.String():        LocationGreenville,
	LocationHamlet.String():            LocationHamlet,
	LocationHavelock.String():          LocationHavelock,
	LocationHenderson.String():         LocationHenderson,
	LocationHendersonville.String():    LocationHendersonville,
	LocationHickory.String():           LocationHickory,
	LocationHighPoint.String():         LocationHighPoint,
	LocationHillsborough.String():      LocationHillsborough,
	LocationHudson.String():            LocationHudson,
	LocationHuntersville.String():      LocationHuntersville,
	LocationJacksonville.String():      LocationJacksonville,
	LocationJefferson.String():         LocationJefferson,
	LocationKernersville.String():      LocationKernersville,
	LocationKinston.String():           LocationKinston,
	LocationLexington.String():         LocationLexington,
	LocationLincolnton.String():        LocationLincolnton,
	LocationLouisburg.String():         LocationLouisburg,
	LocationLumberton.String():         LocationLumberton,
	LocationMarion.String():            LocationMarion,
	LocationMarshall.String():          LocationMarshall,
	LocationMocksville.String():        LocationMocksville,
	LocationMonroe.String():            LocationMonroe,
	LocationMooresville.String():       LocationMooresville,
	LocationMoreheadCity.String():      LocationMoreheadCity,
	LocationMorganton.String():         LocationMorganton,
	LocationMountAiry.String():         LocationMountAiry,
	LocationMountHolly.String():        LocationMountHolly,
	LocationNagsHead.String():          LocationNagsHead,
	LocationNewBern.String():           LocationNewBern,
	LocationNewton.String():            LocationNewton,
	LocationOxford.String():            LocationOxford,
	LocationPolkton.String():           LocationPolkton,
	LocationRaeford.String():           LocationRaeford,
	LocationRaleighEast.String():       LocationRaleighEast,
	LocationRaleighNorth.String():      LocationRaleighNorth,
	LocationRaleighWest.String():       LocationRaleighWest,
	LocationRoanokeRapids.String():     LocationRoanokeRapids,
	LocationRoxboro.String():           LocationRoxboro,
	LocationSalisbury.String():         LocationSalisbury,
	LocationSanford.String():           LocationSanford,
	LocationShallotte.String():         LocationShallotte,
	LocationShelby.String():            LocationShelby,
	LocationSilerCity.String():         LocationSilerCity,
	LocationSmithfield.String():        LocationSmithfield,
	LocationStatesville.String():       LocationStatesville,
	LocationStedman.String():           LocationStedman,
	LocationSylva.String():             LocationSylva,
	LocationTarboro.String():           LocationTarboro,
	LocationTaylorsville.String():      LocationTaylorsville,
	LocationThomasville.String():       LocationThomasville,
	LocationTroy.String():              LocationTroy,
	LocationWashington.String():        LocationWashington,
	LocationWendell.String():           LocationWendell,
	LocationWentworth.String():         LocationWentworth,
	LocationWhiteville.String():        LocationWhiteville,
	LocationWilkesboro.String():        LocationWilkesboro,
	LocationWilliamston.String():       LocationWilliamston,
	LocationWilmingtonNorth.String():   LocationWilmingtonNorth,
	LocationWilmingtonSouth.String():   LocationWilmingtonSouth,
	LocationWilson.String():            LocationWilson,
	LocationWinstonSalemNorth.String(): LocationWinstonSalemNorth,
	LocationWinstonSalemSouth.String(): LocationWinstonSalemSouth,
	LocationYadkinville.String():       LocationYadkinville,
}

func StringToLocation(k string) Location {
	if v, ok := locationMap[k]; !ok {
		return LocationInvalid
	} else {
		return v
	}
}

func ValidLocations() []string {
	return mapToKeys(locationMap)
}
