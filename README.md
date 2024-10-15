Room Reservation App
A desktop-based room reservation system built using Fyne, written in Go. This application allows users to reserve rooms, manage schedules, and administer the system. It includes authentication, undo/redo functionality, and a floor plan view for easy navigation.

Features
Room Reservation: Users can book rooms with time slots and purposes.
User Authentication: Supports user registration, login, and role-based access (Admin and User roles).
Floor Plan View: Visualize and interact with rooms via an uploaded floor plan.
Admin Panel: Admin users can manage rooms, upload floor plans, and manage users.
Undo/Redo: Supports undo and redo actions for room reservations.
Custom Theme: Includes a custom theme for the app's appearance.
Installation
Prerequisites
Go 1.16 or later
Fyne 2.0 or later
Clone the Repository
bash
Copy code
git clone https://github.com/cardoza1991/roomy.git
cd room-reservation-app
Install Dependencies
bash
Copy code
go mod tidy
Run the App
bash
Copy code
go run main.go
Usage
Booking a Room
Log in or register an account.
Select a room from the sidebar or the floor plan view.
Choose a time slot and purpose, then confirm the booking.
Admin Features
Add Rooms: Admins can add new rooms via the Admin Panel.
Upload Floor Plan: Admins can upload a custom floor plan for room selection.
Manage Users: Admins can manage registered users (feature to be implemented).
Undo/Redo
Undo (Ctrl+Z): Reverts the most recent reservation action.
Redo (Ctrl+Y): Re-applies the most recently undone action.
File Storage
reservations.json: Stores room reservations.
users.json: Stores user accounts.
floorplan.png: The uploaded floor plan used in the app.
Customization
The app includes a custom theme (theme/customtheme.go). You can modify the theme for a personalized look and feel.
License
MIT License. See the LICENSE file for more details.

Contributing
Feel free to submit issues or pull requests to contribute to this project.
