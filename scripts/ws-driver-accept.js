#!/usr/bin/env node
/**
 * WebSocket client для отправки driver ride_response
 * Usage: node ws-driver-accept.js <token> <ride_id>
 */

const WebSocket = require('ws');

if (process.argv.length < 4) {
  console.error('Usage: ws-driver-accept.js <driver_token> <ride_id>');
  process.exit(1);
}

const token = process.argv[2];
const rideId = process.argv[3];
const wsUrl = 'ws://localhost:3001/ws';

const ws = new WebSocket(wsUrl);

ws.on('open', () => {
  console.error('WebSocket connected');
  
  // 1. Отправляем auth
  const authMsg = JSON.stringify({ token });
  console.error('Sending auth:', authMsg);
  ws.send(authMsg);
  
  // 2. Через 500ms отправляем ride_response
  setTimeout(() => {
    const rideResponse = JSON.stringify({
      type: 'ride_response',
      data: {
        ride_id: rideId,
        accepted: true,
        current_location: {
          latitude: 43.238949,
          longitude: 76.889709
        }
      }
    });
    console.error('Sending ride_response:', rideResponse);
    ws.send(rideResponse);
    
    // Закрываем через 1 секунду
    setTimeout(() => {
      console.log('Driver accepted ride:', rideId);
      ws.close();
      process.exit(0);
    }, 1000);
  }, 500);
});

ws.on('message', (data) => {
  console.error('< Received:', data.toString());
});

ws.on('error', (err) => {
  console.error('WebSocket error:', err.message);
  process.exit(1);
});

ws.on('close', () => {
  console.error('WebSocket closed');
});

// Timeout safety
setTimeout(() => {
  console.error('Timeout!');
  ws.close();
  process.exit(1);
}, 5000);
