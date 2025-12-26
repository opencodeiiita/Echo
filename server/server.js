/**
 * =============================================================================
 * Echo Server - WebSocket Chat Server
 * =============================================================================
 * 
 * Purpose:
 *   A real-time WebSocket server that handles multiple client connections,
 *   manages usernames, and broadcasts join/leave messages to all connected
 *   clients.
 * 
 * Technologies Used:
 *   - Node.js (runtime environment)
 *   - ws (WebSocket library for Node.js)
 * 
 * How to Run:
 *   1. Navigate to the /server directory
 *   2. Install dependencies: npm install
 *   3. Start the server: node server.js
 *   4. The server will listen on port 8080 (or PORT environment variable)
 * 
 * =============================================================================
 */

// -----------------------------------------------------------------------------
// Dependencies
// -----------------------------------------------------------------------------
const WebSocket = require('ws');

// -----------------------------------------------------------------------------
// Server Configuration
// -----------------------------------------------------------------------------
const PORT = process.env.PORT || 8080;

// -----------------------------------------------------------------------------
// Server Setup
// Create a new WebSocket server instance that listens on the specified port
// -----------------------------------------------------------------------------
const wss = new WebSocket.Server({ port: PORT });

/**
 * Store for connected clients
 * Maps WebSocket connections to their associated usernames
 * This allows us to track who is connected and their identity
 */
const clients = new Map();

// -----------------------------------------------------------------------------
// Utility Functions
// -----------------------------------------------------------------------------

/**
 * Generates a formatted timestamp string for logging purposes
 * Format: [YYYY-MM-DD HH:MM:SS]
 * @returns {string} Formatted timestamp string
 */
function getTimestamp() {
    const now = new Date();
    const year = now.getFullYear();
    const month = String(now.getMonth() + 1).padStart(2, '0');
    const day = String(now.getDate()).padStart(2, '0');
    const hours = String(now.getHours()).padStart(2, '0');
    const minutes = String(now.getMinutes()).padStart(2, '0');
    const seconds = String(now.getSeconds()).padStart(2, '0');
    
    return `[${year}-${month}-${day} ${hours}:${minutes}:${seconds}]`;
}

/**
 * Logs a message to the console with a timestamp
 * @param {string} message - The message to log
 */
function log(message) {
    console.log(`${getTimestamp()} ${message}`);
}

/**
 * Broadcasts a message to ALL connected clients
 * Iterates through all clients in the WebSocket server and sends the message
 * Only sends to clients that are in OPEN state (ready to receive messages)
 * @param {string} message - The message to broadcast to all clients
 */
function broadcast(message) {
    wss.clients.forEach((client) => {
        if (client.readyState === WebSocket.OPEN) {
            client.send(message);
        }
    });
}

// -----------------------------------------------------------------------------
// Connection Handling
// -----------------------------------------------------------------------------

/**
 * Event listener for new client connections
 * When a client connects, we wait for their first message which should be
 * their username, then set up message and close handlers
 */
wss.on('connection', (ws, request) => {
    // Log the new connection (username not yet known)
    const clientIP = request.socket.remoteAddress || 'unknown';
    log(`New connection from ${clientIP}`);
    
    /**
     * Flag to track if the client has sent their username yet
     * The first message from the client is treated as their username
     */
    let hasUsername = false;
    
    // -------------------------------------------------------------------------
    // Message Handling
    // -------------------------------------------------------------------------
    
    /**
     * Event listener for incoming messages from this client
     * The first message is treated as the username
     * Subsequent messages can be handled as needed (e.g., chat messages)
     */
    ws.on('message', (data) => {
        // Convert the received data to a string
        const message = data.toString().trim();
        
        // =====================================================================
        // Username Handling
        // The FIRST message from a client is treated as their username
        // =====================================================================
        if (!hasUsername) {
            // Validate the username (must not be empty)
            if (message.length === 0) {
                ws.send('Error: Username cannot be empty. Please send a valid username.');
                return;
            }
            
            // Store the username associated with this WebSocket connection
            const username = message;
            clients.set(ws, username);
            hasUsername = true;
            
            // Log the successful registration
            log(`User registered: "${username}" from ${clientIP}`);
            
            // =================================================================
            // Broadcasting Logic - User Join Notification
            // Notify ALL connected clients that a new user has joined
            // =================================================================
            const joinMessage = `${username} has joined`;
            broadcast(joinMessage);
            
            // Log the broadcast
            log(`Broadcast: ${joinMessage}`);
            
            return;
        }
        
        // Handle subsequent messages (optional: implement chat functionality)
        // For now, we log received messages but don't broadcast them
        const username = clients.get(ws);
        log(`Message from "${username}": ${message}`);
    });
    
    // -------------------------------------------------------------------------
    // Disconnection Handling
    // -------------------------------------------------------------------------
    
    /**
     * Event listener for client disconnection
     * When a client closes their connection, we notify all other clients
     * and clean up our stored data for that client
     */
    ws.on('close', () => {
        // Get the username of the disconnecting client
        const username = clients.get(ws);
        
        if (username) {
            // Log the disconnection with timestamp
            log(`User disconnected: "${username}"`);
            
            // =================================================================
            // Broadcasting Logic - User Leave Notification
            // Notify ALL connected clients that a user has left
            // =================================================================
            const leaveMessage = `${username} has disconnected`;
            broadcast(leaveMessage);
            
            // Log the broadcast
            log(`Broadcast: ${leaveMessage}`);
            
            // Clean up: Remove the client from our tracking Map
            clients.delete(ws);
        } else {
            // Client disconnected before sending a username
            log(`Anonymous connection closed from ${clientIP}`);
        }
    });
    
    // -------------------------------------------------------------------------
    // Error Handling
    // -------------------------------------------------------------------------
    
    /**
     * Event listener for WebSocket errors
     * Log any errors that occur on this connection
     */
    ws.on('error', (error) => {
        const username = clients.get(ws) || 'Unknown';
        log(`Error for user "${username}": ${error.message}`);
    });
});

// -----------------------------------------------------------------------------
// Server Error Handling
// -----------------------------------------------------------------------------

/**
 * Event listener for server-level errors
 * Handles errors that occur on the WebSocket server itself
 */
wss.on('error', (error) => {
    log(`Server error: ${error.message}`);
});

// -----------------------------------------------------------------------------
// Server Startup
// -----------------------------------------------------------------------------

/**
 * Log server startup message
 * Confirms that the server is running and listening for connections
 */
log(`Echo WebSocket Server started on port ${PORT}`);
log('Waiting for client connections...');

// -----------------------------------------------------------------------------
// Graceful Shutdown
// -----------------------------------------------------------------------------

/**
 * Handle process termination signals for graceful shutdown
 * Closes all client connections and the server properly
 */
process.on('SIGINT', () => {
    log('Shutting down server...');
    
    // Notify all connected clients about server shutdown
    broadcast('Server is shutting down...');
    
    // Close all client connections
    wss.clients.forEach((client) => {
        client.close();
    });
    
    // Close the WebSocket server
    wss.close(() => {
        log('Server shut down complete.');
        process.exit(0);
    });
});
