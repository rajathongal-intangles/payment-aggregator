import { PaymentClient } from './client';
import { processPayment, logPaymentSimple, PaymentEvent } from './logger';
import { config } from './config';

// Track statistics
let stats = {
  existing: 0,
  new: 0,
  total: 0,
  startTime: Date.now(),
};

async function main() {
  console.log(`
╔═══════════════════════════════════════════════════════════════╗
║           PAYMENT gRPC CLIENT (Node.js)                       ║
╠═══════════════════════════════════════════════════════════════╣
║  Connecting to: ${config.grpc.address.padEnd(43)}║
║  Proto file:    ${config.proto.path.slice(-43).padEnd(43)}║
╚═══════════════════════════════════════════════════════════════╝
`);

  const client = new PaymentClient();

  try {
    // Connect to server
    await client.connect();

    // Subscribe to payment stream
    console.log('\n📡 Subscribing to payment stream...\n');
    console.log('─'.repeat(65));

    const stream = client.streamPayments(
      // On each payment (one at a time!)
      (event: PaymentEvent) => {
        stats.total++;
        
        if (event.eventType === 'new') {
          stats.new++;
        } else {
          stats.existing++;
        }

        // Use detailed or simple logging
        // processPayment(event);  // Detailed box format
        logPaymentSimple(event);   // One-line format
      },
      
      // On error
      (err: Error) => {
        console.error('\n❌ Stream error:', err.message);
        printStats();
        process.exit(1);
      },
      
      // On stream end
      () => {
        console.log('\n📪 Stream ended by server');
        printStats();
      }
    );

    // Handle graceful shutdown
    process.on('SIGINT', () => {
      console.log('\n\n⏳ Shutting down...');
      stream.cancel();
      client.close();
      printStats();
      process.exit(0);
    });

    process.on('SIGTERM', () => {
      stream.cancel();
      client.close();
      process.exit(0);
    });

    // Keep process running
    console.log('Waiting for payments... (Ctrl+C to stop)\n');

  } catch (err) {
    console.error('❌ Failed to start client:', err);
    process.exit(1);
  }
}

function printStats() {
  const duration = ((Date.now() - stats.startTime) / 1000).toFixed(1);
  console.log(`
┌─────────────────────────────────────────────────────────────┐
│                      SESSION STATS                          │
├─────────────────────────────────────────────────────────────┤
│  Total Payments:    ${String(stats.total).padEnd(38)}│
│  Existing:          ${String(stats.existing).padEnd(38)}│
│  New:               ${String(stats.new).padEnd(38)}│
│  Duration:          ${(duration + 's').padEnd(38)}│
└─────────────────────────────────────────────────────────────┘`);
}

// Run
main();