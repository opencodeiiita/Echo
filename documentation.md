This is a temporary documentation file to document your features.
Don't overwrite this file, only append to it.

In here, after solving an issue successfully, **(IF AND ONLY IF IT IS MENTIONED IN THE NOTES)** , document everything you've added/changed.

Format of documentation: </br>

{your github id} {issue number and name}: </br>

- List of all features you've implemented, any particular format regarding them etc. Screenshots are allowed and appreciated.
- also - zero emojis. or em dashes.
- Begin writing documentation from after the heading.

# DOCUMENTATION

### {LooninS} {#58 Time to mine}

- Added chat.go, changed main.go to call chat.go
- chat.go has 3 functions:

```go
connectToEchoServer() // connects to the server, sends the username, printout the server message
getUsername() // gets the username from the user/stdin and returns it as a string
getTimestamp() // returns the current timestamp as a string in the format of "02/01/2006 03:04:05 PM"
```

- go.mod was updated to include the module github.com/gorilla/websocket
- tested the code locally

### {HarshitRSethi} {#67 Getting an Upgrade!}

- Updated server/server.js so that:
  - the first message sent by a client is treated as the username and stored for that connection
  - every message after that is treated as a chat message
  - each message is broadcast to all connected clients
  - messages include a timestamp and the username of the sender
- Updated client/chat.go so that:
  - after sending the username once, the user can continue sending chat messages from the terminal
  - messages received from the server are printed along with timestamps
  - a goroutine is used to listen for incoming messages while the main thread handles user input
- Tested locally with two clients connected to the same server to verify that messages are sent and received correctly, with usernames and timestamps shown as expected

### {Abhineshhh} {#71 Add user prompt for messages}

- Added "Enter Message : " prompt in client/chat.go message input loop
- Fixed server/server.js: moved message handler inside username callback to prevent username broadcast as chat message
- Fixed client/chat.go: implemented ANSI escape codes to reprint prompt after incoming messages for cleaner display

### {IIT2023139} {#79 Feat: Implement Initial TUI Design, Chat Interactivity, and Channel Navigation.}

- Designed a complete TUI interface using Bubble Tea and Lipgloss
- Implemented Login View with input fields for Server, Username, and Password
- Created Chat View with vertical sidebar for channels and message viewport
- Added navigation logic for cycling through channels and inputs
- ensured correct alignment of UI elements and clean message formatting

### {sshekhar563} {#103 Acquire Hardware}

- Connected server to MongoDB using mongoose
- Added User model (server/models/User.js) with fields: username, connectedAt, isOnline
- Added Message model (server/models/Message.js) with fields: sender, content, timestamp
- Server logs every user connection to the database
- Server checks if username is already taken (online), sends error message and disconnects if so
- Server logs every chat message with sender and timestamp to the database
- Users are marked offline when they disconnect
- Added .env.example with MongoDB Atlas connection string template
- Added dotenv and mongoose dependencies to package.json

### {Krishna200608} {#104 Eye Spy}

- Updated client/chat.go:
  - Added getServerAddress() function to capture server URL from user input via stdin.
  - Implemented default fallback to "localhost:8080" if input is empty.
- Updated client/main.go:
  - Removed dependency on command-line flags for server address.
  - Reordered logic to prompt for Server Address first, then Username.
- Tested locally: Verified connection sequence with localhost server.

### {Krishna200608} {#108 Into Fire (Client ver.)}

- Updated client/chat.go:

  - Added getPassword() function to capture user password from stdin.
  - Updated connectToEchoServer() signature to accept password parameter.
  - Modified message listener loop to remove "Server -> Client" prefix and redundant client-side timestamp, aligning output with the required spec.

- Updated client/main.go:
  - Added call to getPassword() after username prompt.
  - Passed password to connectToEchoServer().
- Tested locally: Verified password prompt appears and chat messages display cleanly.

### {dwivediprashant} {#111 Into Fire (Server ver.)}

- Updated server/server.js:

  - Added bcrypt dependency for password hashing.
  - Implemented authentication functions: hashPassword(), verifyPassword(), findUser(), createUser().
  - Modified connection logic to handle JSON authentication data.
  - Added proper error handling for wrong passwords and duplicate online users.

- Updated server/models/User.js:

  - Added password field to user schema for storing hashed passwords.

- Updated server/package.json:

  - Added bcrypt dependency for password hashing.

- Updated client/chat.go:

  - Added JSON encoding for authentication data.
  - Modified connectToEchoServer() to send {"username": "...", "password": "..."}.
  - Added input validation to prevent empty usernames and passwords.
  - Implemented proper authentication error handling with clean exit on failure.
  - Added success message display for correct authentication.

- Tested locally: Verified new user registration, existing user login, wrong password rejection, and proper error handling.

### {Krishna200608} {#127 Ooh, Shiny! (Implementation ver.)}

Built a complete TUI for Echo using Bubble Tea and Lipgloss.

**Files:** `config.go`, `styles.go`, `tui_model.go`, `theme.conf`, `main.go`, `chat.go`

**Login Screen:** ASCII logo in accent color, animated input fields with glow effects, password toggle, pulsing connect button, styled error messages.

**Chat Screen:** Header with ECHO branding, online indicator, username, session timer. Scrollable message viewport with bubbles. Dynamic textarea (expands to 5 lines). Animated input border.

**Controls:** `Enter` send, `Alt+Enter` newline, `PageUp/Down` scroll, `Ctrl+U` clear, `Tab` navigate, `Esc` quit.

**Theme System:** 10 presets via `THEME: N` in theme.conf: Default, Cyberpunk, Forest, Ocean, Sunset, Dracula, Nord, Monokai, Gruvbox, Tokyo Night. All UI elements use theme colors dynamically.

**Animations:** Pulsing dots, glowing borders, progress bar, adaptive timing.

Tested locally.

### {Krishna200608} {#132 You've got a friend in me}

Implemented private messaging with whisper commands.

**Server Changes (`server.js`):**
- Added `!whisper <user> <msg>` and `!w <user> <msg>` command parsing
- Private messages sent only to sender and receiver
- Error response if target user not online

**Client Changes:**
- `config.go`: Added `PrivMsgColor` to Config struct and all 15 theme presets
- `styles.go`: Added `PrivMsg` style and `PrivMsgColor` for rendering
- `tui_model.go`: Updated `ChatMessage` with `IsPrivate` field, `parseMessage()` detects "privately said:" format, `renderMessages()` shows `[WHISPER]` label in distinct color
- `theme.conf`: Added `PRIV_MESSAGE` key for custom color

**Usage:** Type `!whisper username message` or `!w username message` to send private messages.

**Visual:** Private messages display with `[WHISPER]` prefix in theme-specific color (pink/purple variants).

Tested locally with multiple clients.