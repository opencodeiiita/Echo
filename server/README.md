### Echo Server

The backend server for the Echo application, built with Node.js and WebSockets.

#### Prerequisites

- Node.js (v14 or higher)
- npm

#### Installation

1. Navigate to the `server` directory.
2. Install dependencies:
   ```bash
   npm install
   ```

#### Usage

Start the development server:

```bash
npm run dev
```

Start the production server:

```bash
npm start
```

The server runs on port `8080` by default. You can configure this via a `.env` file (see below).

#### Architecture

- **Protocol**: Raw WebSockets
- **Connection Logic**:
  1. Client connects to `ws://localhost:8080`.
  2. Client **MUST** send a plain text string immediately as their first message. This is registered as the **username**.
  3. Server broadcasts `<username> has joined` to all connected clients.
  4. Connections and disconnections are logged to the console with timestamps.

#### Environment Variables

Create a `.env` file in the `server` directory:

```ini
PORT=8080
```
