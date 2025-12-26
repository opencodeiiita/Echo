/**
 * Echo Server - Real-time Communication Server
 * 
 * This server handles WebSocket connections for the Echo application,
 * enabling real-time messaging between connected clients.
 * 
 * Features:
 * - WebSocket-based real-time communication using Socket.IO
 * - User connection/disconnection management
 * - Username registration and broadcasting
 * - Timestamped logging of all events
 * - RESTful API endpoints for health checks
 */

const express = require('express');
const http = require('http');
const socketIO = require('socket.io');
const cors = require('cors');


const app = express();
const server = http.createServer(app);


const io = socketIO(server, {
  cors: {
    origin: '*', 
    methods: ['GET', 'POST']
  }
});


app.use(cors());
app.use(express.json());


const connectedUsers = new Map();

/**
 * Utility function to get current timestamp in ISO format
 * @returns {string} Current timestamp
 */
function getTimestamp() {
  return new Date().toISOString();
}

/**
 * Utility function to log with timestamp
 * @param {string} message - Message to log
 */
function logWithTimestamp(message) {
  console.log(`[${getTimestamp()}] ${message}`);
}

/**
 * Get list of all connected usernames
 * @returns {Array<string>} Array of connected usernames
 */
function getConnectedUsernames() {
  return Array.from(connectedUsers.values());
}


app.get('/', (req, res) => {
  res.json({
    status: 'running',
    message: 'Echo Server is running',
    timestamp: getTimestamp(),
    connectedUsers: connectedUsers.size,
    uptime: process.uptime()
  });
});


app.get('/health', (req, res) => {
  res.json({
    status: 'healthy',
    timestamp: getTimestamp(),
    connectedUsers: connectedUsers.size
  });
});


app.get('/users', (req, res) => {
  res.json({
    users: getConnectedUsernames(),
    count: connectedUsers.size,
    timestamp: getTimestamp()
  });
});

io.on('connection', (socket) => {
  logWithTimestamp(`New client connected. Socket ID: ${socket.id}`);
  

  let usernameSet = false;

  socket.on('register-username', (data) => {
    if (usernameSet) {
      socket.emit('error', { message: 'Username already set for this connection' });
      return;
    }

    const username = data.username || data;
    
    if (!username || typeof username !== 'string' || username.trim() === '') {
      socket.emit('error', { message: 'Invalid username provided' });
      return;
    }

    const trimmedUsername = username.trim();

    const existingUsernames = Array.from(connectedUsers.values());
    if (existingUsernames.includes(trimmedUsername)) {
      socket.emit('error', { message: 'Username already taken' });
      return;
    }

   
    connectedUsers.set(socket.id, trimmedUsername);
    usernameSet = true;

    logWithTimestamp(`User "${trimmedUsername}" registered (Socket ID: ${socket.id})`);

    
    socket.emit('registration-success', {
      username: trimmedUsername,
      timestamp: getTimestamp()
    });


    socket.broadcast.emit('user-joined', {
      username: trimmedUsername,
      timestamp: getTimestamp(),
      message: `${trimmedUsername} has joined the chat`
    });

   
    socket.emit('user-list', {
      users: getConnectedUsernames(),
      timestamp: getTimestamp()
    });
  });

  
  socket.on('message', (data) => {
    const username = connectedUsers.get(socket.id);
    
    if (!username) {
      socket.emit('error', { message: 'Please register a username first' });
      return;
    }

    const message = data.message || data;
    logWithTimestamp(`Message from ${username}: ${message}`);


    io.emit('message', {
      username: username,
      message: message,
      timestamp: getTimestamp()
    });
  });

 
  socket.on('disconnect', () => {
    const username = connectedUsers.get(socket.id);

    if (username) {
      logWithTimestamp(`User "${username}" disconnected (Socket ID: ${socket.id})`);

   
      connectedUsers.delete(socket.id);

      
      socket.broadcast.emit('user-left', {
        username: username,
        timestamp: getTimestamp(),
        message: `${username} has left the chat`
      });
    } else {
      logWithTimestamp(`Anonymous client disconnected (Socket ID: ${socket.id})`);
    }
  });

 
  socket.on('error', (error) => {
    logWithTimestamp(`Socket error for ${socket.id}: ${error.message}`);
  });
});



const PORT = process.env.PORT || 3000;

server.listen(PORT, () => {
  logWithTimestamp(`Echo Server started successfully`);
  logWithTimestamp(`Server listening on port ${PORT}`);
  logWithTimestamp(`WebSocket endpoint: ws://localhost:${PORT}`);
  logWithTimestamp(`HTTP endpoint: http://localhost:${PORT}`);
  console.log('\n==============================================');
  console.log('Echo Server is ready to accept connections!');
  console.log('==============================================\n');
});


process.on('SIGINT', () => {
  logWithTimestamp('Received SIGINT. Shutting down gracefully...');
  
  
  io.emit('server-shutdown', {
    message: 'Server is shutting down',
    timestamp: getTimestamp()
  });

  
  server.close(() => {
    logWithTimestamp('Server closed. All connections terminated.');
    process.exit(0);
  });

  
  setTimeout(() => {
    logWithTimestamp('Forcing shutdown...');
    process.exit(1);
  }, 5000);
});

process.on('SIGTERM', () => {
  logWithTimestamp('Received SIGTERM. Shutting down gracefully...');
  server.close(() => {
    logWithTimestamp('Server closed.');
    process.exit(0);
  });
});

module.exports = { app, server, io };