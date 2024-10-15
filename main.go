// main.go

package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/color"
	"io"
	"log"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	customtheme "roomy/theme" // Ensure this path is correct

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"golang.org/x/crypto/bcrypt"
)

// Command interface for undo/redo functionality
type Command interface {
	Execute()
	Undo()
}

// Define data structures and variables
type Reservation struct {
	RoomName  string
	Date      string
	StartTime time.Time
	EndTime   time.Time
	Purpose   string
	Leader    string
	Student   string
	Priority  int
	Active    bool // For soft delete
}

type Room struct {
	Name         string
	Reservations []Reservation
	mu           sync.Mutex
	Position     fyne.Position // For floor plan
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

const (
	timeLayout12Hour   = "3:04 PM"
	floorPlanImagePath = "floorplan.png" // Path to the uploaded floor plan image
)

func getPriority(purpose string) int {
	// Implement priority logic if needed
	return 0 // Placeholder
}

// Implement methods for Room
func (r *Room) Reserve(reservation Reservation) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Check for overlapping reservations
	for _, res := range r.Reservations {
		if res.Active && res.Date == reservation.Date && (reservation.StartTime.Before(res.EndTime) && reservation.EndTime.After(res.StartTime)) {
			return fmt.Errorf("time slot already reserved")
		}
	}

	reservation.Active = true
	r.Reservations = append(r.Reservations, reservation)
	saveReservations()

	return nil
}

func (r *Room) DeleteReservation(index int) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if index >= 0 && index < len(r.Reservations) {
		r.Reservations[index].Active = false // Soft delete
		saveReservations()
	}
}

// User authentication
type User struct {
	Username     string
	PasswordHash []byte
	Role         string
}

var users []User
var currentUser *User

func createUser(username, password, role string) error {
	if len(password) < 8 {
		return fmt.Errorf("password must be at least 8 characters long")
	}

	for _, user := range users {
		if user.Username == username {
			return fmt.Errorf("username already exists")
		}
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	users = append(users, User{
		Username:     username,
		PasswordHash: passwordHash,
		Role:         role,
	})
	saveUsers()
	return nil
}

func authenticateUser(username, password string) (*User, error) {
	for i, user := range users {
		if user.Username == username {
			err := bcrypt.CompareHashAndPassword(user.PasswordHash, []byte(password))
			if err != nil {
				return nil, fmt.Errorf("incorrect password")
			}
			return &users[i], nil
		}
	}
	return nil, fmt.Errorf("user not found")
}

func showRegistration(content *fyne.Container, w fyne.Window) {
	usernameEntry := widget.NewEntry()
	passwordEntry := widget.NewPasswordEntry()
	confirmPasswordEntry := widget.NewPasswordEntry()

	form := dialog.NewForm("Register", "Register", "Cancel", []*widget.FormItem{
		{Text: "Username", Widget: usernameEntry},
		{Text: "Password", Widget: passwordEntry},
		{Text: "Confirm Password", Widget: confirmPasswordEntry},
	}, func(confirmed bool) {
		if confirmed {
			if passwordEntry.Text != confirmPasswordEntry.Text {
				dialog.ShowError(errors.New("passwords do not match"), w)
				return
			}
			err := createUser(usernameEntry.Text, passwordEntry.Text, "User")
			if err != nil {
				dialog.ShowError(err, w)
			} else {
				dialog.ShowInformation("Success", "User registered successfully", w)
			}
		}
	}, w)
	form.Show()
}

func showLogin(content *fyne.Container, w fyne.Window, onSuccess func(*User)) {
	usernameEntry := widget.NewEntry()
	passwordEntry := widget.NewPasswordEntry()

	form := dialog.NewForm("Login", "Login", "Cancel", []*widget.FormItem{
		{Text: "Username", Widget: usernameEntry},
		{Text: "Password", Widget: passwordEntry},
	}, func(confirmed bool) {
		if confirmed {
			user, err := authenticateUser(usernameEntry.Text, passwordEntry.Text)
			if err != nil {
				dialog.ShowError(err, w)
			} else {
				currentUser = user
				onSuccess(user)
			}
		}
	}, w)
	form.Show()
}

// Undo/Redo functionality
var undoStack []Command
var redoStack []Command

// ReservationCommand for undo/redo
type ReservationCommand struct {
	reservation Reservation
	room        *Room
	index       int // Index in the room's reservation slice
}

func (c *ReservationCommand) Execute() {
	c.room.Reserve(c.reservation)
	c.index = len(c.room.Reservations) - 1
}

func (c *ReservationCommand) Undo() {
	c.room.DeleteReservation(c.index)
}

func main() {
	a := app.NewWithID("com.example.roomreservation")
	a.Settings().SetTheme(&customtheme.CustomTheme{})
	w := a.NewWindow("Room Booking")

	// Load reservations and users
	loadReservations()
	loadUsers(w) // Pass 'w' here

	// Create initial content
	content := container.NewMax()
	content.Objects = []fyne.CanvasObject{widget.NewLabel("Please log in to continue.")}

	// Create sidebar
	sidebar := createSidebar(content, w)

	// Main layout
	mainLayout := container.NewBorder(nil, nil, sidebar, nil, content)

	// Implement global keyboard shortcuts
	w.Canvas().AddShortcut(&desktop.CustomShortcut{
		KeyName:  fyne.KeyZ,
		Modifier: fyne.KeyModifierControl,
	}, func(shortcut fyne.Shortcut) {
		undo()
		content.Refresh()
	})

	w.Canvas().AddShortcut(&desktop.CustomShortcut{
		KeyName:  fyne.KeyY,
		Modifier: fyne.KeyModifierControl,
	}, func(shortcut fyne.Shortcut) {
		redo()
		content.Refresh()
	})

	w.SetContent(mainLayout)
	w.Resize(fyne.NewSize(1024, 768))
	w.ShowAndRun()
}

func createSidebar(content *fyne.Container, w fyne.Window) *fyne.Container {
	reservationViewsButton := widget.NewButtonWithIcon("Reservation Views", theme.ContentCopyIcon(), func() {
		interval := 1 * time.Hour // Hourly intervals
		content.Objects = []fyne.CanvasObject{createGridScheduleView(content, interval, w)}
		content.Refresh()
	})

	floorPlanButton := widget.NewButtonWithIcon("Floor Plan View", theme.NavigateNextIcon(), func() {
		content.Objects = []fyne.CanvasObject{createFloorPlanView(w)}
		content.Refresh()
	})

	adminButton := widget.NewButtonWithIcon("Admin Panel", theme.SettingsIcon(), func() {
		if currentUser != nil && currentUser.Role == "Admin" {
			showAdminTab(content, w)
		} else {
			dialog.ShowInformation("Access Denied", "You do not have permission to access this feature.", w)
		}
	})

	buttons := []*widget.Button{reservationViewsButton, floorPlanButton}

	if currentUser != nil {
		logoutButton := widget.NewButtonWithIcon("Logout", theme.LogoutIcon(), func() {
			currentUser = nil
			content.Objects = []fyne.CanvasObject{widget.NewLabel("Please log in to continue.")}
			content.Refresh()
		})
		buttons = append(buttons, logoutButton)
		if currentUser.Role == "Admin" {
			buttons = append(buttons, adminButton)
		}
	} else {
		loginButton := widget.NewButtonWithIcon("Login", theme.LoginIcon(), func() {
			showLogin(content, w, func(user *User) {
				currentUser = user
				interval := 1 * time.Hour // Hourly intervals
				content.Objects = []fyne.CanvasObject{createGridScheduleView(content, interval, w)}
				content.Refresh()
			})
		})
		registerButton := widget.NewButtonWithIcon("Register", theme.DocumentCreateIcon(), func() {
			showRegistration(content, w)
		})
		buttons = append(buttons, loginButton, registerButton)
	}

	// Apply styling to sidebar buttons
	for _, btn := range buttons {
		btn.Importance = widget.HighImportance
	}

	sidebar := container.NewVBox()
	for _, btn := range buttons {
		sidebar.Add(btn)
	}
	return sidebar
}

func NewTappableImage(img image.Image) *TappableImage {
	tappableImage := &TappableImage{
		Image: canvas.NewImageFromImage(img),
	}
	tappableImage.Image.FillMode = canvas.ImageFillContain // Set FillMode here
	tappableImage.ExtendBaseWidget(tappableImage)
	return tappableImage
}

// Floor plan view
func createFloorPlanView(w fyne.Window) fyne.CanvasObject {
	// Load the floor plan image
	imgFile, err := os.Open(floorPlanImagePath)
	if err != nil {
		return widget.NewLabel("Floor plan not uploaded.")
	}
	defer imgFile.Close()

	img, _, err := image.Decode(imgFile)
	if err != nil {
		dialog.ShowError(err, w)
		return widget.NewLabel("Error loading floor plan image.")
	}

	// Here you call NewTappableImage to create your image with the correct FillMode
	floorPlanImage := NewTappableImage(img)

	floorPlan := container.NewWithoutLayout(floorPlanImage)
	// Add room icons to the floor plan
	for _, room := range rooms {
		roomCopy := room // Capture variable for closure
		roomButton := widget.NewButton(room.Name, func() {
			// Handle room booking from floor plan
			openRoomBooking(roomCopy, w)
		})
		// Position the button
		roomButton.Move(room.Position)
		floorPlan.Add(roomButton)
	}

	// If admin, allow placing rooms on the floor plan
	if currentUser != nil && currentUser.Role == "Admin" {
		floorPlanImage.OnTapped = func(event *fyne.PointEvent) {
			// Show a dialog to select a room to place
			roomNames := []string{}
			for _, room := range rooms {
				roomNames = append(roomNames, room.Name)
			}
			roomSelect := widget.NewSelect(roomNames, func(selected string) {
				// Update the room's position
				for _, room := range rooms {
					if room.Name == selected {
						room.Position = event.Position
						saveReservations()
						// Refresh the content
						content := createFloorPlanView(w)
						w.SetContent(content)
						break
					}
				}
			})
			dialog.ShowCustom("Select Room", "Close", roomSelect, w)
		}
	}

	scroll := container.NewScroll(floorPlan)
	return scroll
}

// Define TappableImage
// Define TappableImage
type TappableImage struct {
	widget.BaseWidget
	Image    *canvas.Image
	OnTapped func(*fyne.PointEvent)
}

func (t *TappableImage) Tapped(event *fyne.PointEvent) {
	if t.OnTapped != nil {
		t.OnTapped(event)
	}
}

func (t *TappableImage) TappedSecondary(event *fyne.PointEvent) {}

// Implement Widget interface
func (t *TappableImage) CreateRenderer() fyne.WidgetRenderer {
	return &tappableImageRenderer{
		image: t.Image,
	}
}

type tappableImageRenderer struct {
	image *canvas.Image
}

func (r *tappableImageRenderer) Layout(size fyne.Size) {
	r.image.Resize(size)
}

func (r *tappableImageRenderer) MinSize() fyne.Size {
	return r.image.MinSize()
}

func (r *tappableImageRenderer) Refresh() {
	r.image.Refresh()
}

func (r *tappableImageRenderer) BackgroundColor() color.Color {
	return color.Transparent
}

func (r *tappableImageRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{r.image}
}

func (r *tappableImageRenderer) Destroy() {}

// Implement openRoomBooking function
func openRoomBooking(room *Room, w fyne.Window) {
	// Implement room booking from floor plan
	dialog.ShowInformation("Room Booking", fmt.Sprintf("Booking for room: %s", room.Name), w)
}

// Implement createGridScheduleView
func createGridScheduleView(content *fyne.Container, interval time.Duration, w fyne.Window) fyne.CanvasObject {
	today := time.Now().Format("2006-01-02")
	timeSlots := generateTimeSlots(interval)
	grid := container.NewGridWithRows(len(timeSlots) + 1)

	selectedSlots := make(map[string]*ColorButton)

	// Header row with room names
	header := container.NewGridWithColumns(len(rooms) + 1)
	header.Add(widget.NewLabel("Time Slots"))
	for _, room := range rooms {
		header.Add(widget.NewLabel(room.Name))
	}
	grid.Add(header)

	// Generate grid rows for each time slot
	for _, slot := range timeSlots {
		slotCopy := slot // capture variable
		row := container.NewGridWithColumns(len(rooms) + 1)
		row.Add(widget.NewLabel(slot))

		for _, room := range rooms {
			roomCopy := room // capture variable
			reserved := checkRoomReservation(roomCopy, today, slotCopy)
			button := NewColorButton("", nil)
			button.Disable()

			if reserved {
				button.Text = "Booked"
				button.BackgroundColor = color.NRGBA{R: 220, G: 53, B: 69, A: 255} // Danger color
				button.Refresh()
			} else {
				// Make the button selectable
				button.Enable()
				button.Text = "" // Keep the button text empty
				// Copy variables for closure
				roomNameCopy := roomCopy.Name
				slotTimeCopy := slotCopy

				button.OnTapped = func() {
					handleSlotSelection(selectedSlots, roomNameCopy, slotTimeCopy, button, interval, w)
				}
				button.Refresh()
			}
			row.Add(button)
		}
		grid.Add(row)
	}

	// Button to confirm selection and open the reservation form
	confirmButton := widget.NewButton("Book Now", func() {
		if len(selectedSlots) == 0 {
			dialog.ShowInformation("Error", "Please select room(s) and time slot(s).", w)
			return
		}

		// Group selected slots by room
		roomSlotsMap := make(map[string][]string)
		for key := range selectedSlots {
			roomName, slot := parseSlotKey(key)
			roomSlotsMap[roomName] = append(roomSlotsMap[roomName], slot)
		}

		// For simplicity, we'll only allow booking one room at a time
		if len(roomSlotsMap) > 1 {
			dialog.ShowInformation("Error", "Please select time slots for only one room at a time.", w)
			return
		}

		var roomName string
		var slots []string
		for rn, sl := range roomSlotsMap {
			roomName = rn
			slots = sl
			break
		}

		// Sort slots by time
		sort.Slice(slots, func(i, j int) bool {
			t1, _ := time.Parse(timeLayout12Hour, slots[i])
			t2, _ := time.Parse(timeLayout12Hour, slots[j])
			return t1.Before(t2)
		})

		// Check if slots are contiguous
		if !areSlotsContiguous(slots, interval) {
			dialog.ShowInformation("Error", "Please select contiguous time slots.", w)
			return
		}

		// Get start and end times
		startTimeStr := slots[0]
		lastSlot := slots[len(slots)-1]
		endTimeStr := incrementTimeSlot(lastSlot, interval)

		// Open reservation form with pre-filled data
		openReservationForm(content, roomName, today, startTimeStr, endTimeStr, interval, w)
	})

	// Adjust the button's appearance
	confirmButton.Importance = widget.HighImportance

	scroll := container.NewVScroll(grid)
	scroll.SetMinSize(fyne.NewSize(800, 600))

	// Wrap the confirm button in an HBox to prevent it from stretching
	buttonContainer := container.NewHBox(layout.NewSpacer(), confirmButton, layout.NewSpacer())

	// Use container.NewBorder to place the button at the bottom without stretching
	return container.NewBorder(nil, buttonContainer, nil, nil, scroll)
}

// Handle slot selection logic
func handleSlotSelection(selectedSlots map[string]*ColorButton, roomNameCopy, slotTimeCopy string, button *ColorButton, interval time.Duration, w fyne.Window) {
	slotKey := fmt.Sprintf("%s_%s", roomNameCopy, slotTimeCopy)
	if _, exists := selectedSlots[slotKey]; exists {
		delete(selectedSlots, slotKey)
		button.BackgroundColor = customtheme.ButtonColor
		button.Refresh()
	} else {
		// Ensure only one room's slots are selected at a time
		for key := range selectedSlots {
			existingRoomName, _ := parseSlotKey(key)
			if existingRoomName != roomNameCopy {
				dialog.ShowInformation("Error", "Please select time slots for only one room at a time.", w)
				return
			}
		}
		selectedSlots[slotKey] = button
		button.BackgroundColor = color.NRGBA{R: 40, G: 167, B: 69, A: 255} // Success green
		button.Refresh()
	}
}

// Implement other necessary functions
func parseSlotKey(key string) (roomName, timeSlot string) {
	parts := strings.SplitN(key, "_", 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return "", ""
}

func incrementTimeSlot(timeSlot string, interval time.Duration) string {
	t, err := time.Parse(timeLayout12Hour, timeSlot)
	if err != nil {
		return timeSlot
	}
	t = t.Add(interval)
	return t.Format(timeLayout12Hour)
}

func areSlotsContiguous(slots []string, interval time.Duration) bool {
	if len(slots) <= 1 {
		return true
	}
	for i := 1; i < len(slots); i++ {
		prevTime, err1 := time.Parse(timeLayout12Hour, slots[i-1])
		currTime, err2 := time.Parse(timeLayout12Hour, slots[i])
		if err1 != nil || err2 != nil {
			return false
		}
		if currTime.Sub(prevTime) != interval {
			return false
		}
	}
	return true
}

func generateTimeSlots(interval time.Duration) []string {
	var slots []string
	start := time.Date(0, 0, 0, 8, 0, 0, 0, time.UTC) // Start at 8 AM
	end := time.Date(0, 0, 0, 23, 0, 0, 0, time.UTC)  // End at 11 PM
	for t := start; t.Before(end) || t.Equal(end); t = t.Add(interval) {
		slots = append(slots, t.Format("3:04 PM"))
	}
	return slots
}

func checkRoomReservation(room *Room, date, timeSlot string) bool {
	room.mu.Lock()
	defer room.mu.Unlock()
	slotTime, err := time.Parse(timeLayout12Hour, timeSlot)
	if err != nil {
		return false
	}
	for _, res := range room.Reservations {
		if res.Active && res.Date == date {
			if slotTime.Equal(res.StartTime) || (slotTime.After(res.StartTime) && slotTime.Before(res.EndTime)) {
				return true
			}
		}
	}
	return false
}

func openReservationForm(content *fyne.Container, roomName, date, startTimeStr, endTimeStr string, interval time.Duration, w fyne.Window) {
	purposeSelect := widget.NewSelect([]string{
		"Meeting",
		"Study Session",
		"Presentation",
		"Other",
	}, func(value string) {})
	purposeSelect.PlaceHolder = "Select Purpose"

	leaderEntry := widget.NewEntry()
	leaderEntry.SetPlaceHolder("Your Name")
	studentEntry := widget.NewEntry()
	studentEntry.SetPlaceHolder("Additional Info")

	form := &widget.Form{
		Items: []*widget.FormItem{
			{Text: "Room:", Widget: widget.NewLabel(roomName)},
			{Text: "Date:", Widget: widget.NewLabel(date)},
			{Text: "Start Time:", Widget: widget.NewLabel(startTimeStr)},
			{Text: "End Time:", Widget: widget.NewLabel(endTimeStr)},
			{Text: "Purpose:", Widget: purposeSelect},
			{Text: "Your Name:", Widget: leaderEntry},
			{Text: "Additional Info:", Widget: studentEntry},
		},
		OnSubmit: func() {
			purpose := purposeSelect.Selected
			leader := leaderEntry.Text
			student := studentEntry.Text

			if purpose == "" {
				dialog.ShowError(errors.New("please select a purpose"), w)
				return
			}
			if leader == "" {
				dialog.ShowError(errors.New("please enter your name"), w)
				return
			}

			var room *Room
			for _, r := range rooms {
				if r.Name == roomName {
					room = r
					break
				}
			}
			if room == nil {
				dialog.ShowError(errors.New("room not found"), w)
				return
			}

			// Parse date and time
			startTime, err := time.Parse(timeLayout12Hour, startTimeStr)
			if err != nil {
				dialog.ShowError(errors.New("invalid start time format"), w)
				return
			}

			endTime, err := time.Parse(timeLayout12Hour, endTimeStr)
			if err != nil {
				dialog.ShowError(errors.New("invalid end time format"), w)
				return
			}

			// Combine date with time
			dateOnly, _ := time.Parse("2006-01-02", date)
			startDateTime := time.Date(dateOnly.Year(), dateOnly.Month(), dateOnly.Day(), startTime.Hour(), startTime.Minute(), 0, 0, time.Local)
			endDateTime := time.Date(dateOnly.Year(), dateOnly.Month(), dateOnly.Day(), endTime.Hour(), endTime.Minute(), 0, 0, time.Local)

			if startDateTime.After(endDateTime) {
				dialog.ShowError(errors.New("end time cannot be before start time"), w)
				return
			}

			reservation := Reservation{
				RoomName:  roomName,
				Date:      date,
				StartTime: startDateTime,
				EndTime:   endDateTime,
				Purpose:   purpose,
				Leader:    leader,
				Student:   student,
				Priority:  getPriority(purpose),
				Active:    true,
			}

			// Show booking confirmation
			showBookingConfirmation(reservation, w, func() {
				// Proceed with reservation using Command pattern
				cmd := &ReservationCommand{
					reservation: reservation,
					room:        room,
				}
				err := cmd.room.Reserve(cmd.reservation)
				if err != nil {
					dialog.ShowError(err, w)
				} else {
					undoStack = append(undoStack, cmd)
					// Clear redo stack
					redoStack = []Command{}
					dialog.ShowInformation("Success", fmt.Sprintf("Room '%s' has been reserved on %s from %s to %s.", roomName, date, startTimeStr, endTimeStr), w)
					// Refresh the grid view
					content.Objects = []fyne.CanvasObject{createGridScheduleView(content, interval, w)}
					content.Refresh()
				}
			})
		},
	}

	dialog.ShowCustom("Make Reservation", "Close", container.NewVBox(form), w)
}

func showBookingConfirmation(reservation Reservation, w fyne.Window, onConfirm func()) {
	content := widget.NewLabel(fmt.Sprintf(
		"Room: %s\nDate: %s\nTime: %s - %s\nPurpose: %s\nName: %s\nInfo: %s",
		reservation.RoomName,
		reservation.Date,
		reservation.StartTime.Format("3:04 PM"),
		reservation.EndTime.Format("3:04 PM"),
		reservation.Purpose,
		reservation.Leader,
		reservation.Student,
	))
	dialog.ShowCustomConfirm("Confirm Booking", "Confirm", "Cancel", content, func(confirmed bool) {
		if confirmed {
			onConfirm()
		}
	}, w)
}

// Undo and Redo functions
func undo() {
	if len(undoStack) == 0 {
		return
	}
	cmd := undoStack[len(undoStack)-1]
	undoStack = undoStack[:len(undoStack)-1]
	cmd.Undo()
	redoStack = append(redoStack, cmd)
}

func redo() {
	if len(redoStack) == 0 {
		return
	}
	cmd := redoStack[len(redoStack)-1]
	redoStack = redoStack[:len(redoStack)-1]
	cmd.Execute()
	undoStack = append(undoStack, cmd)
}

// Load and save reservations
func loadReservations() {
	file, err := os.Open("reservations.json")
	if os.IsNotExist(err) {
		log.Println("reservations.json file not found, creating a new one.")
		saveReservations()
		return
	} else if err != nil {
		log.Printf("Error opening reservations file: %v\n", err)
		return
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	err = decoder.Decode(&rooms)
	if err != nil {
		log.Printf("Error decoding reservations: %v\n", err)
		return
	}

	// Initialize Reservations slice if it is nil
	for _, room := range rooms {
		if room.Reservations == nil {
			room.Reservations = []Reservation{}
		}
	}
}

func saveReservations() {
	file, err := os.Create("reservations.json")
	if err != nil {
		log.Printf("Error saving reservations: %v\n", err)
		return
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	err = encoder.Encode(&rooms)
	if err != nil {
		log.Printf("Error encoding reservations: %v\n", err)
	}
}

// Load and save users
func loadUsers(w fyne.Window) {
	file, err := os.Open("users.json")
	if os.IsNotExist(err) {
		log.Println("users.json file not found, creating a new one.")
		// Since no user exists, prompt admin creation
		dialog.ShowInformation("First-time setup", "No admin found. Please create an admin account.", w)
		showAdminRegistration(nil, w) // Show admin registration form
		return
	} else if err != nil {
		log.Printf("Error opening users file: %v\n", err)
		return
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	err = decoder.Decode(&users)
	if err != nil {
		log.Printf("Error decoding users: %v\n", err)
		return
	}

	// Check if any admin exists
	adminExists := false
	for _, user := range users {
		if user.Role == "Admin" {
			adminExists = true
			break
		}
	}

	if !adminExists {
		dialog.ShowInformation("No Admin Account", "There are no admin accounts. Please create an admin account.", w)
		showAdminRegistration(nil, w) // Show admin registration form if no admin exists
	}
}

func showAdminRegistration(content *fyne.Container, w fyne.Window) {
	usernameEntry := widget.NewEntry()
	passwordEntry := widget.NewPasswordEntry()
	confirmPasswordEntry := widget.NewPasswordEntry()

	form := dialog.NewForm("Admin Registration", "Register", "Cancel", []*widget.FormItem{
		{Text: "Username", Widget: usernameEntry},
		{Text: "Password", Widget: passwordEntry},
		{Text: "Confirm Password", Widget: confirmPasswordEntry},
	}, func(confirmed bool) {
		if confirmed {
			if passwordEntry.Text != confirmPasswordEntry.Text {
				dialog.ShowError(errors.New("passwords do not match"), w)
				return
			}
			err := createUser(usernameEntry.Text, passwordEntry.Text, "Admin")
			if err != nil {
				dialog.ShowError(err, w)
			} else {
				dialog.ShowInformation("Success", "Admin account created successfully", w)
			}
		}
	}, w)
	form.Show()
}

func saveUsers() {
	file, err := os.Create("users.json")
	if err != nil {
		log.Printf("Error saving users: %v\n", err)
		return
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	err = encoder.Encode(&users)
	if err != nil {
		log.Printf("Error encoding users: %v\n", err)
	}
}

// Implement Admin Panel
func showAdminTab(content *fyne.Container, w fyne.Window) {
	if currentUser == nil || currentUser.Role != "Admin" {
		dialog.ShowInformation("Access Denied", "You do not have permission to access this feature.", w)
		return
	}
	content.Objects = []fyne.CanvasObject{createAdminPanel(content, w)}
	content.Refresh()
}

func createAdminPanel(content *fyne.Container, w fyne.Window) fyne.CanvasObject {
	addRoomButton := widget.NewButton("Add Room", func() {
		roomNameEntry := widget.NewEntry()
		form := dialog.NewForm("Add Room", "Add", "Cancel", []*widget.FormItem{
			{Text: "Room Name", Widget: roomNameEntry},
		}, func(confirm bool) {
			if confirm {
				roomName := roomNameEntry.Text
				if roomName == "" {
					dialog.ShowError(errors.New("room name cannot be empty"), w)
					return
				}
				addRoom(roomName, w)
				content.Objects = []fyne.CanvasObject{createAdminPanel(content, w)}
				content.Refresh()
			}
		}, w)
		form.Resize(fyne.NewSize(400, 200))
		form.Show()
	})

	manageUsersButton := widget.NewButton("Manage Users", func() {
		manageUsers(w)
	})

	uploadFloorPlanButton := widget.NewButton("Upload Floor Plan", func() {
		uploadFloorPlan(w)
	})

	settingsButton := widget.NewButton("Settings", func() {
		showSettings(w)
	})

	return container.NewVBox(
		addRoomButton,
		manageUsersButton,
		uploadFloorPlanButton,
		settingsButton,
	)
}

func addRoom(name string, w fyne.Window) {
	rooms = append(rooms, &Room{Name: name})
	saveReservations()
	dialog.ShowInformation("Room Added", fmt.Sprintf("Room '%s' has been successfully added.", name), w)
}

func manageUsers(w fyne.Window) {
	// Implement user management UI
	dialog.ShowInformation("Manage Users", "User management is not implemented yet.", w)
}

func uploadFloorPlan(w fyne.Window) {
	fileDialog := dialog.NewFileOpen(func(reader fyne.URIReadCloser, err error) {
		if err != nil || reader == nil {
			return
		}
		defer reader.Close()

		// Save the uploaded image
		data, err := io.ReadAll(reader)
		if err != nil {
			dialog.ShowError(err, w)
			return
		}
		err = os.WriteFile(floorPlanImagePath, data, 0644)
		if err != nil {
			dialog.ShowError(err, w)
			return
		}
		dialog.ShowInformation("Success", "Floor plan uploaded successfully.", w)
	}, w)
	fileDialog.SetFilter(storage.NewExtensionFileFilter([]string{".png", ".jpg", ".jpeg"}))
	fileDialog.Show()
}

func showSettings(w fyne.Window) {
	// Implement settings UI
	dialog.ShowInformation("Settings", "Settings are not implemented yet.", w)
}

// Custom ColorButton with enhancements
type ColorButton struct {
	widget.BaseWidget
	Text            string
	OnTapped        func()
	BackgroundColor color.Color
	Disabled        bool
}

func NewColorButton(text string, tapped func()) *ColorButton {
	btn := &ColorButton{
		Text:            text,
		OnTapped:        tapped,
		BackgroundColor: customtheme.ButtonColor, // Use custom theme color
	}
	btn.ExtendBaseWidget(btn)
	return btn
}

func (b *ColorButton) CreateRenderer() fyne.WidgetRenderer {
	label := canvas.NewText(b.Text, customtheme.TextColor) // Use custom text color
	label.Alignment = fyne.TextAlignCenter

	background := canvas.NewRectangle(b.BackgroundColor)

	objects := []fyne.CanvasObject{background, label}

	return &colorButtonRenderer{
		button:     b,
		label:      label,
		background: background,
		objects:    objects,
	}
}

type colorButtonRenderer struct {
	button     *ColorButton
	label      *canvas.Text
	background *canvas.Rectangle
	objects    []fyne.CanvasObject
}

func (r *colorButtonRenderer) Layout(size fyne.Size) {
	r.background.Resize(size)
	r.label.Move(fyne.NewPos(customtheme.Padding, customtheme.Padding))
	r.label.Resize(size.Subtract(fyne.NewSize(customtheme.Padding*2, customtheme.Padding*2)))
}

func (r *colorButtonRenderer) MinSize() fyne.Size {
	labelSize := r.label.MinSize()
	padding := customtheme.Padding
	return fyne.NewSize(labelSize.Width+padding*2, labelSize.Height+padding*2)
}

func (r *colorButtonRenderer) Refresh() {
	r.label.Text = r.button.Text
	if r.button.Disabled {
		r.label.Color = customtheme.DisabledTextColor
	} else {
		r.label.Color = customtheme.TextColor
	}
	r.label.Refresh()
	r.background.FillColor = r.button.BackgroundColor
	r.background.Refresh()
}

func (r *colorButtonRenderer) BackgroundColor() color.Color {
	return color.Transparent
}

func (r *colorButtonRenderer) Objects() []fyne.CanvasObject {
	return r.objects
}

func (r *colorButtonRenderer) Destroy() {}

func (b *ColorButton) Tapped(_ *fyne.PointEvent) {
	if b.Disabled {
		return
	}
	if b.OnTapped != nil {
		b.OnTapped()
	}
}

func (b *ColorButton) TappedSecondary(_ *fyne.PointEvent) {}

func (b *ColorButton) Disable() {
	b.Disabled = true
	b.Refresh()
}

func (b *ColorButton) Enable() {
	b.Disabled = false
	b.Refresh()
}

func (b *ColorButton) IsDisabled() bool {
	return b.Disabled
}
