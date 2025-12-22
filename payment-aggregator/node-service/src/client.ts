import * as grpc from '@grpc/grpc-js';
import * as protoLoader from '@grpc/proto-loader';
import path from 'path';
import { config } from './config';
import { PaymentEvent } from './logger';

// Proto loader options
const PROTO_OPTIONS: protoLoader.Options = {
  keepCase: false,      // Convert to camelCase
  longs: String,        // Convert int64 to string
  enums: Number,        // Keep enums as numbers
  defaults: true,       // Include default values
  oneofs: true,
};

// Load proto definition
const packageDefinition = protoLoader.loadSync(
  config.proto.path,
  PROTO_OPTIONS
);

const protoDescriptor = grpc.loadPackageDefinition(packageDefinition) as any;
const PaymentService = protoDescriptor.payment.PaymentService;

export class PaymentClient {
  private client: any;
  private connected: boolean = false;

  constructor() {
    this.client = new PaymentService(
      config.grpc.address,
      grpc.credentials.createInsecure()
    );
  }

  /**
   * Wait for the client to be ready
   */
  async connect(timeoutMs: number = 5000): Promise<void> {
    return new Promise((resolve, reject) => {
      const deadline = Date.now() + timeoutMs;
      
      this.client.waitForReady(deadline, (err: Error | undefined) => {
        if (err) {
          reject(new Error(`Failed to connect to gRPC server: ${err.message}`));
        } else {
          this.connected = true;
          console.log(`✅ Connected to gRPC server at ${config.grpc.address}`);
          resolve();
        }
      });
    });
  }

  /**
   * Get a single payment by ID
   */
  async getPayment(paymentId: string): Promise<any> {
    return new Promise((resolve, reject) => {
      this.client.getPayment({ paymentId }, (err: Error, response: any) => {
        if (err) {
          reject(err);
        } else {
          resolve(response);
        }
      });
    });
  }

  /**
   * List payments with optional filters
   */
  async listPayments(options: {
    provider?: number;
    status?: number;
    limit?: number;
  } = {}): Promise<any> {
    return new Promise((resolve, reject) => {
      this.client.listPayments(
        {
          provider: options.provider || 0,
          status: options.status || 0,
          limit: options.limit || 10,
        },
        (err: Error, response: any) => {
          if (err) {
            reject(err);
          } else {
            resolve(response);
          }
        }
      );
    });
  }

  /**
   * Subscribe to payment stream - receives payments ONE AT A TIME
   */
  streamPayments(
    onPayment: (event: PaymentEvent) => void,
    onError?: (err: Error) => void,
    onEnd?: () => void,
    options: { provider?: number } = {}
  ): grpc.ClientReadableStream<PaymentEvent> {
    
    const stream = this.client.streamPayments({
      provider: options.provider || 0,
      status: 0,
      limit: 0,
    });

    stream.on('data', (event: PaymentEvent) => {
      onPayment(event);
    });

    stream.on('error', (err: Error) => {
      if (onError) {
        onError(err);
      } else {
        console.error('❌ Stream error:', err.message);
      }
    });

    stream.on('end', () => {
      if (onEnd) {
        onEnd();
      } else {
        console.log('📪 Stream ended');
      }
    });

    return stream;
  }

  /**
   * Close the client connection
   */
  close(): void {
    this.client.close();
    this.connected = false;
    console.log('👋 Disconnected from gRPC server');
  }
}