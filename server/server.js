const WebSocket = require('ws');
const http = require('http');

const PORT = 8080;

const server = http.createServer((req, res) => {
    if (req.url === '/health') {
        res.writeHead(200, { 'Content-Type': 'application/json' });
        res.end(JSON.stringify({ status: 'ok' }));
        return;
    }
    res.writeHead(404);
    res.end();
});

const wss = new WebSocket.Server({ server });

wss.on('connection', (ws, req) => {
    const ip = req.socket.remoteAddress;
    console.log(`[${new Date().toISOString()}] Connection from ${ip}`);

    let username = null;

    ws.on('message', (msg) => {
        const text = msg.toString().trim();
        if (!username) {
            if (!text) return ws.send('Error: Empty username');
            
            username = text;
            console.log(`[${new Date().toISOString()}] User joined: ${username}`);
            ws.send(`Welcome, ${username}!`);
            
            // Broadcast join
            wss.clients.forEach(c => {
                if (c.readyState === WebSocket.OPEN) c.send(`${username} joined.`);
            });
        }
    });

    ws.on('close', () => {
        if (username) console.log(`[${new Date().toISOString()}] User left: ${username}`);
    });
});

server.listen(PORT, () => console.log(`Echo running on :${PORT}`));
