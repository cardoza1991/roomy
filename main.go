package main

import (
	"fmt"
	"image/color"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type Reservation struct {
	Date      string
	StartTime time.Time
	EndTime   time.Time
	Purpose   string
	Leader    string
	Student   string
	Priority  int
}

type Room struct {
	Name         string
	Reservations []Reservation
	mu           sync.Mutex
}

var rooms = []*Room{
	{Name: "Study Room 1"},
	{Name: "Study Room 2"},
	{Name: "Study Room 3"},
	{Name: "Study Room 4"},
	{Name: "Study Room 5"},
	{Name: "Conference Room"},
	{Name: "LRE Room"},
}

var leaders = []string{"Leader 1", "Leader 2", "Leader 3"}

const timeLayout12Hour = "03:04 PM"

func getPriority(purpose string) int {
	switch purpose {
	case "1 = Unbaptized Contact In-person":
		return 1
	case "2 = Baptized Persecuted Member In-person":
		return 2
	case "3 = Baptized Member In-person":
		return 3
	case "4 = Unbaptized Contact Zoom":
		return 4
	case "5 = Baptized Member Zoom":
		return 5
	case "6 = Group Activities":
		return 6
	case "7 = Team Activities":
		return 7
	default:
		return 0
	}
}

func (r *Room) Reserve(reservation Reservation) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for i := len(r.Reservations) - 1; i >= 0; i-- {
		res := r.Reservations[i]
		if res.Date == reservation.Date && !(reservation.EndTime.Before(res.StartTime) || reservation.StartTime.After(res.EndTime)) {
			if reservation.Priority < res.Priority {
				fyne.CurrentApp().SendNotification(&fyne.Notification{
					Title:   "Reservation Overridden",
					Content: fmt.Sprintf("Your reservation for %s on %s from %s to %s has been overridden by a higher priority booking. Please notify the group leader.", res.Purpose, res.Date, res.StartTime.Format(timeLayout12Hour), res.EndTime.Format(timeLayout12Hour)),
				})
				r.Reservations = append(r.Reservations[:i], r.Reservations[i+1:]...)
			} else {
				return fmt.Errorf("time slot already reserved by a higher or equal priority reservation")
			}
		}
	}

	r.Reservations = append(r.Reservations, reservation)
	return nil
}

func (r *Room) ListReservations(date string) []Reservation {
	r.mu.Lock()
	defer r.mu.Unlock()

	var reservations []Reservation
	for _, res := range r.Reservations {
		if res.Date == date {
			reservations = append(reservations, res)
		}
	}
	return reservations
}

func (r *Room) DeleteReservation(index int) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if index >= 0 && index < len(r.Reservations) {
		r.Reservations = append(r.Reservations[:index], r.Reservations[index+1:]...)
	}
}

func addRoom(name string) {
	rooms = append(rooms, &Room{Name: name})
}

func removeRoom(name string) {
	for i, room := range rooms {
		if room.Name == name {
			rooms = append(rooms[:i], rooms[i+1:]...)
			break
		}
	}
}

func createRoomList(content *fyne.Container) *widget.List {
	list := widget.NewList(
		func() int {
			return len(rooms)
		},
		func() fyne.CanvasObject {
			label := widget.NewLabel("")
			label.TextStyle = fyne.TextStyle{Bold: true}
			return label
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			obj.(*widget.Label).SetText(rooms[id].Name)
		},
	)
	list.OnSelected = func(id widget.ListItemID) {
		selectedRoom := rooms[id].Name
		content.Objects = []fyne.CanvasObject{createRoomDetailView(selectedRoom)}
		content.Refresh()
	}
	return list
}

func createRoomAvailability(room *Room) fyne.CanvasObject {
	grid := container.NewGridWithColumns(1)

	timeSlots := []string{
		"12:00 AM", "01:00 AM", "02:00 AM", "03:00 AM", "04:00 AM", "05:00 AM", "06:00 AM", "07:00 AM",
		"08:00 AM", "09:00 AM", "10:00 AM", "11:00 AM", "12:00 PM", "01:00 PM", "02:00 PM", "03:00 PM",
		"04:00 PM", "05:00 PM", "06:00 PM", "07:00 PM", "08:00 PM", "09:00 PM", "10:00 PM", "11:00 PM",
	}

	for _, slot := range timeSlots {
		slotContainer := container.NewHBox(
			widget.NewLabel(slot),
			canvas.NewRectangle(color.RGBA{0, 255, 0, 255}),
		)
		grid.Add(slotContainer)
	}

	for _, res := range room.Reservations {
		startIndex := findTimeSlotIndex(res.StartTime.Format(timeLayout12Hour))
		endIndex := findTimeSlotIndex(res.EndTime.Format(timeLayout12Hour))

		for i := startIndex; i <= endIndex; i++ {
			slotRect := canvas.NewRectangle(color.RGBA{255, 0, 0, 255})
			slotRect.SetMinSize(fyne.NewSize(150, 20))
			grid.Objects[i] = container.NewHBox(
				widget.NewLabel(timeSlots[i]),
				slotRect,
			)
			animateRectangle(slotRect)
		}
	}

	return grid
}

func animateRectangle(rect *canvas.Rectangle) {
	fromColor := color.RGBA{0, 255, 0, 255}
	toColor := color.RGBA{255, 0, 0, 255}
	anim := canvas.NewColorRGBAAnimation(fromColor, toColor, time.Second*2, func(c color.Color) {
		rect.FillColor = c
		canvas.Refresh(rect)
	})
	anim.AutoReverse = true
	anim.RepeatCount = fyne.AnimationRepeatForever
	anim.Start()
}

func findTimeSlotIndex(timeStr string) int {
	timeSlots := []string{
		"12:00 AM", "01:00 AM", "02:00 AM", "03:00 AM", "04:00 AM", "05:00 AM", "06:00 AM", "07:00 AM",
		"08:00 AM", "09:00 AM", "10:00 AM", "11:00 AM", "12:00 PM", "01:00 PM", "02:00 PM", "03:00 PM",
		"04:00 PM", "05:00 PM", "06:00 PM", "07:00 PM", "08:00 PM", "09:00 PM", "10:00 PM", "11:00 PM",
	}

	for i, slot := range timeSlots {
		if slot == timeStr {
			return i
		}
	}
	return -1
}

func searchAvailableRooms(date string, startTime, endTime time.Time) []*Room {
	var availableRooms []*Room
	for _, room := range rooms {
		available := true
		for _, res := range room.Reservations {
			if res.Date == date && (startTime.Before(res.EndTime) && endTime.After(res.StartTime)) {
				available = false
				break
			}
		}
		if available {
			availableRooms = append(availableRooms, room)
		}
	}
	return availableRooms
}

func createCalendar(setDate func(string)) fyne.CanvasObject {
	calendar := container.NewGridWithColumns(7)
	currentMonth := time.Now().Month()
	currentYear := time.Now().Year()

	updateCalendar := func(month time.Month, year int) {
		calendar.Objects = []fyne.CanvasObject{}
		firstDay := time.Date(year, month, 1, 0, 0, 0, 0, time.Local)
		lastDay := firstDay.AddDate(0, 1, -1)

		daysOfWeek := []string{"Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"}
		for _, day := range daysOfWeek {
			calendar.Add(widget.NewLabel(day))
		}

		for i := 0; i < int(firstDay.Weekday()); i++ {
			calendar.Add(widget.NewLabel(" "))
		}

		for day := 1; day <= lastDay.Day(); day++ {
			d := day // capture range variable
			dateButton := widget.NewButton(fmt.Sprintf("%d", day), func() {
				selectedDate := time.Date(year, month, d, 0, 0, 0, 0, time.Local)
				setDate(selectedDate.Format("2006-01-02"))
			})
			calendar.Add(dateButton)
		}
	}

	updateCalendar(currentMonth, currentYear)

	prevButton := widget.NewButton("<", func() {
		if currentMonth == time.January {
			currentMonth = time.December
			currentYear--
		} else {
			currentMonth--
		}
		updateCalendar(currentMonth, currentYear)
	})

	nextButton := widget.NewButton(">", func() {
		if currentMonth == time.December {
			currentMonth = time.January
			currentYear++
		} else {
			currentMonth++
		}
		updateCalendar(currentMonth, currentYear)
	})

	return container.NewVBox(
		container.NewHBox(prevButton, widget.NewLabel(fmt.Sprintf("%s %d", currentMonth, currentYear)), nextButton),
		calendar,
	)
}

func createRoomDetailView(roomName string) fyne.CanvasObject {
	var room *Room
	for _, r := range rooms {
		if r.Name == roomName {
			room = r
			break
		}
	}
	if room == nil {
		return widget.NewLabel("Room not found")
	}

	reservationList := container.NewVBox()

	for i, res := range room.Reservations {
		i := i // capture range variable
		resText := fmt.Sprintf("%s %s to %s - %s, %s, %s", res.Date, res.StartTime.Format(timeLayout12Hour), res.EndTime.Format(timeLayout12Hour), res.Purpose, res.Leader, res.Student)
		reservationItem := container.NewHBox(
			widget.NewLabel(resText),
			widget.NewButton("Delete", func() {
				passwordEntry := widget.NewEntry()
				passwordEntry.SetPlaceHolder("Enter password")
				passwordEntry.Password = true

				confirmDialog := dialog.NewCustomConfirm("Confirm Action", "Confirm", "Cancel", passwordEntry, func(confirmed bool) {
					if confirmed && passwordEntry.Text == "1948" {
						room.DeleteReservation(i)
						reservationList.Objects = createReservationList(room)
						reservationList.Refresh()
						fyne.CurrentApp().SendNotification(&fyne.Notification{
							Title:   "Success",
							Content: "Reservation has been deleted.",
						})
					} else if confirmed {
						fyne.CurrentApp().SendNotification(&fyne.Notification{
							Title:   "Error",
							Content: "Incorrect password.",
						})
					}
				}, fyne.CurrentApp().Driver().AllWindows()[0])
				confirmDialog.Show()
			}),
		)
		reservationList.Add(reservationItem)
	}

	availabilityScroll := container.NewVScroll(createRoomAvailability(room))
	availabilityScroll.SetMinSize(fyne.NewSize(300, 600))

	return container.NewVBox(
		widget.NewLabel(fmt.Sprintf("Room: %s", room.Name)),
		widget.NewLabel("Room Availability:"),
		availabilityScroll,
		reservationList,
	)
}

func createReservationList(room *Room) []fyne.CanvasObject {
	var reservations []fyne.CanvasObject

	for i, res := range room.Reservations {
		i := i // capture range variable
		resText := fmt.Sprintf("%s %s to %s - %s, %s, %s", res.Date, res.StartTime.Format(timeLayout12Hour), res.EndTime.Format(timeLayout12Hour), res.Purpose, res.Leader, res.Student)
		reservationItem := container.NewHBox(
			widget.NewLabel(resText),
			widget.NewButton("Delete", func() {
				passwordEntry := widget.NewEntry()
				passwordEntry.SetPlaceHolder("Enter password")
				passwordEntry.Password = true

				confirmDialog := dialog.NewCustomConfirm("Confirm Action", "Confirm", "Cancel", passwordEntry, func(confirmed bool) {
					if confirmed && passwordEntry.Text == "1948" {
						room.DeleteReservation(i)
						fyne.CurrentApp().SendNotification(&fyne.Notification{
							Title:   "Success",
							Content: "Reservation has been deleted.",
						})
					} else if confirmed {
						fyne.CurrentApp().SendNotification(&fyne.Notification{
							Title:   "Error",
							Content: "Incorrect password.",
						})
					}
				}, fyne.CurrentApp().Driver().AllWindows()[0])
				confirmDialog.Show()
			}),
		)
		reservations = append(reservations, reservationItem)
	}

	return reservations
}

func main() {
	a := app.NewWithID("com.example.roomreservation")
	w := a.NewWindow("Room Reservation System")

	content := container.NewMax()
	sidebar := container.NewVBox(
		widget.NewButtonWithIcon("Room Management", theme.ContentCopyIcon(), func() {
			content.Objects = []fyne.CanvasObject{createRoomList(content)}
			content.Refresh()
		}),
	)

	roomList := createRoomList(content)
	content.Objects = []fyne.CanvasObject{roomList}

	topBar := widget.NewToolbar(
		widget.NewToolbarAction(theme.ContentAddIcon(), func() {
			roomNameEntry := widget.NewEntry()
			passwordEntry := widget.NewPasswordEntry()
			form := dialog.NewForm("Add Room", "Add", "Cancel", []*widget.FormItem{
				{Text: "Room Name", Widget: roomNameEntry},
				{Text: "Password", Widget: passwordEntry},
			}, func(confirm bool) {
				if confirm {
					roomName := roomNameEntry.Text
					password := passwordEntry.Text
					if password == "1948" {
						addRoom(roomName)
						content.Objects = []fyne.CanvasObject{createRoomList(content)}
						content.Refresh()
						fyne.CurrentApp().SendNotification(&fyne.Notification{
							Title:   "Success",
							Content: "Room has been added.",
						})
					} else {
						fyne.CurrentApp().SendNotification(&fyne.Notification{
							Title:   "Error",
							Content: "Incorrect password.",
						})
					}
				}
			}, w)
			form.Resize(fyne.NewSize(400, 200))
			form.Show()
		}),
		widget.NewToolbarAction(theme.ContentRemoveIcon(), func() {
			roomNames := make([]string, len(rooms))
			for i, room := range rooms {
				roomNames[i] = room.Name
			}
			roomSelect := widget.NewSelect(roomNames, func(value string) {})
			passwordEntry := widget.NewPasswordEntry()
			form := dialog.NewForm("Remove Room", "Remove", "Cancel", []*widget.FormItem{
				{Text: "Room Name", Widget: roomSelect},
				{Text: "Password", Widget: passwordEntry},
			}, func(confirm bool) {
				if confirm {
					roomName := roomSelect.Selected
					password := passwordEntry.Text
					if password == "1948" {
						removeRoom(roomName)
						content.Objects = []fyne.CanvasObject{createRoomList(content)}
						content.Refresh()
						fyne.CurrentApp().SendNotification(&fyne.Notification{
							Title:   "Success",
							Content: "Room has been removed.",
						})
					} else {
						fyne.CurrentApp().SendNotification(&fyne.Notification{
							Title:   "Error",
							Content: "Incorrect password.",
						})
					}
				}
			}, w)
			form.Resize(fyne.NewSize(400, 200))
			form.Show()
		}),
		widget.NewToolbarAction(theme.CheckButtonCheckedIcon(), func() {
			roomNames := make([]string, len(rooms))
			for i, room := range rooms {
				roomNames[i] = room.Name
			}
			roomSelect := widget.NewSelect(roomNames, func(value string) {})
			selectedDate := ""

			var calendarButton *widget.Button

			calendarButton = widget.NewButton("Select Date", func() {
				calendarDialog := dialog.NewCustom("Select Date", "Close", createCalendar(func(date string) {
					selectedDate = date
					calendarButton.SetText(date)
				}), w)
				calendarDialog.Resize(fyne.NewSize(300, 300))
				calendarDialog.Show()
			})

			timeSlots := []string{
				"12:00 AM", "01:00 AM", "02:00 AM", "03:00 AM", "04:00 AM", "05:00 AM", "06:00 AM", "07:00 AM",
				"08:00 AM", "09:00 AM", "10:00 AM", "11:00 AM", "12:00 PM", "01:00 PM", "02:00 PM", "03:00 PM",
				"04:00 PM", "05:00 PM", "06:00 PM", "07:00 PM", "08:00 PM", "09:00 PM", "10:00 PM", "11:00 PM",
			}
			startTimeSelect := widget.NewSelect(timeSlots, func(value string) {})
			endTimeSelect := widget.NewSelect(timeSlots, func(value string) {})
			purposeSelect := widget.NewSelect([]string{
				"1 = Unbaptized Contact In-person",
				"2 = Baptized Persecuted Member In-person",
				"3 = Baptized Member In-person",
				"4 = Unbaptized Contact Zoom",
				"5 = Baptized Member Zoom",
				"6 = Group Activities",
				"7 = Team Activities",
			}, func(value string) {})
			leaderSelect := widget.NewSelect(leaders, func(value string) {})
			studentEntry := widget.NewEntry()
			studentEntry.SetPlaceHolder("Student Name")

			form := &widget.Form{
				Items: []*widget.FormItem{
					{Text: "Select Room:", Widget: roomSelect},
					{Text: "Select Date:", Widget: calendarButton},
					{Text: "Select Start Time:", Widget: startTimeSelect},
					{Text: "Select End Time:", Widget: endTimeSelect},
					{Text: "Meeting Type:", Widget: purposeSelect},
					{Text: "Leader Name:", Widget: leaderSelect},
					{Text: "Student Name:", Widget: studentEntry},
				},
				OnSubmit: func() {
					roomName := roomSelect.Selected
					date := selectedDate
					startTimeStr := startTimeSelect.Selected
					endTimeStr := endTimeSelect.Selected
					purpose := purposeSelect.Selected
					leader := leaderSelect.Selected
					student := studentEntry.Text

					var room *Room
					for _, r := range rooms {
						if r.Name == roomName {
							room = r
							break
						}
					}
					if room == nil {
						fyne.CurrentApp().SendNotification(&fyne.Notification{
							Title:   "Error",
							Content: "Room not found",
						})
						return
					}

					startTime, err := time.Parse(timeLayout12Hour, startTimeStr)
					if err != nil {
						fyne.CurrentApp().SendNotification(&fyne.Notification{
							Title:   "Error",
							Content: "Invalid start time format",
						})
						return
					}

					endTime, err := time.Parse(timeLayout12Hour, endTimeStr)
					if err != nil {
						fyne.CurrentApp().SendNotification(&fyne.Notification{
							Title:   "Error",
							Content: "Invalid end time format",
						})
						return
					}

					reservation := Reservation{
						Date:      date,
						StartTime: startTime,
						EndTime:   endTime,
						Purpose:   purpose,
						Leader:    leader,
						Student:   student,
						Priority:  getPriority(purpose),
					}

					if startTime.After(endTime) {
						fyne.CurrentApp().SendNotification(&fyne.Notification{
							Title:   "Error",
							Content: "End time cannot be before start time",
						})
						return
					}

					err = room.Reserve(reservation)
					if err != nil {
						fyne.CurrentApp().SendNotification(&fyne.Notification{
							Title:   "Error",
							Content: err.Error(),
						})
					} else {
						fyne.CurrentApp().SendNotification(&fyne.Notification{
							Title:   "Success",
							Content: fmt.Sprintf("Room '%s' has been reserved on %s from %s to %s.", roomName, date, startTimeStr, endTimeStr),
						})
					}
				},
			}
			dialog.ShowForm("Reserve Room", "Submit", "Cancel", form.Items, func(bool) { form.OnSubmit() }, w)
		}),
		widget.NewToolbarAction(theme.AccountIcon(), func() {
			actionGroup := widget.NewRadioGroup([]string{"Add Leader", "Remove Leader"}, func(value string) {})
			leaderEntry := widget.NewEntry()
			leaderSelect := widget.NewSelect(leaders, func(value string) {})

			form := &widget.Form{
				Items: []*widget.FormItem{
					{Text: "Action:", Widget: actionGroup},
					{Text: "Leader Name (for Add):", Widget: leaderEntry},
					{Text: "Select Leader (for Remove):", Widget: leaderSelect},
				},
				OnSubmit: func() {
					action := actionGroup.Selected
					leaderName := leaderEntry.Text
					selectedLeader := leaderSelect.Selected

					switch action {
					case "Add Leader":
						if leaderName != "" {
							leaders = append(leaders, leaderName)
							fyne.CurrentApp().SendNotification(&fyne.Notification{
								Title:   "Success",
								Content: fmt.Sprintf("Leader '%s' has been added.", leaderName),
							})
						}
					case "Remove Leader":
						for i, leader := range leaders {
							if leader == selectedLeader {
								leaders = append(leaders[:i], leaders[i+1:]...)
								fyne.CurrentApp().SendNotification(&fyne.Notification{
									Title:   "Success",
									Content: fmt.Sprintf("Leader '%s' has been removed.", selectedLeader),
								})
								break
							}
						}
					}
				},
			}
			dialog.ShowForm("Manage Leaders", "Submit", "Cancel", form.Items, func(bool) { form.OnSubmit() }, w)
		}),
		widget.NewToolbarAction(theme.SearchIcon(), func() {
			timeSlots := []string{
				"12:00 AM", "01:00 AM", "02:00 AM", "03:00 AM", "04:00 AM", "05:00 AM", "06:00 AM", "07:00 AM",
				"08:00 AM", "09:00 AM", "10:00 AM", "11:00 AM", "12:00 PM", "01:00 PM", "02:00 PM", "03:00 PM",
				"04:00 PM", "05:00 PM", "06:00 PM", "07:00 PM", "08:00 PM", "09:00 PM", "10:00 PM", "11:00 PM",
			}
			startTimeSelect := widget.NewSelect(timeSlots, func(value string) {})
			endTimeSelect := widget.NewSelect(timeSlots, func(value string) {})
			selectedDate := ""
			var calendarButton *widget.Button
			calendarButton = widget.NewButton("Select Date", func() {
				calendarDialog := dialog.NewCustom("Select Date", "Close", createCalendar(func(date string) {
					selectedDate = date
					calendarButton.SetText(date)
				}), w)
				calendarDialog.Resize(fyne.NewSize(300, 300))
				calendarDialog.Show()
			})

			form := &widget.Form{
				Items: []*widget.FormItem{
					{Text: "Select Date:", Widget: calendarButton},
					{Text: "Select Start Time:", Widget: startTimeSelect},
					{Text: "Select End Time:", Widget: endTimeSelect},
				},
				OnSubmit: func() {
					startTimeStr := startTimeSelect.Selected
					endTimeStr := endTimeSelect.Selected

					startTime, err := time.Parse(timeLayout12Hour, startTimeStr)
					if err != nil {
						fyne.CurrentApp().SendNotification(&fyne.Notification{
							Title:   "Error",
							Content: "Invalid start time format",
						})
						return
					}

					endTime, err := time.Parse(timeLayout12Hour, endTimeStr)
					if err != nil {
						fyne.CurrentApp().SendNotification(&fyne.Notification{
							Title:   "Error",
							Content: "Invalid end time format",
						})
						return
					}

					availableRooms := searchAvailableRooms(selectedDate, startTime, endTime)
					result := "Available Rooms:\n"
					if len(availableRooms) == 0 {
						result += "No rooms available for the selected time slot."
					} else {
						for _, room := range availableRooms {
							result += room.Name + "\n"
						}
					}
					dialog.ShowInformation("Search Results", result, w)
				},
			}
			dialog.ShowForm("Search Available Rooms", "Search", "Cancel", form.Items, func(bool) { form.OnSubmit() }, w)
		}),
	)

	mainLayout := container.NewBorder(topBar, nil, sidebar, nil, content)

	w.SetContent(mainLayout)
	w.Resize(fyne.NewSize(800, 600))
	w.ShowAndRun()
}
