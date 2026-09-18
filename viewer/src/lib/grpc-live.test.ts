// JEH-HOHO generated-client transport regression for the graduated Viewer.
import assert from "node:assert/strict";
import test from "node:test";
import {
  Server,
  ServerCredentials,
  type ServerWritableStream,
} from "@grpc/grpc-js";
import {
  CapitalReservoirCause,
  CapitalReservoirEventType,
  CapitalReservoirEvidence,
  ProductEvidenceEnvelope,
  ProductEvidenceServiceService,
  SubscribeEvidenceResponse,
  type SubscribeEvidenceRequest,
} from "@/generated/jeh_hoho/v1/JEH_HOHO";
import type { CapitalReservoirEvent } from "./capital-reservoir";
import { subscribeToReservoirEvents } from "./grpc-live";

test("generated product stream preserves and normalizes Capital values", async () => {
  const server = new Server();
  server.addService(ProductEvidenceServiceService, {
    subscribeEvidence(call: ServerWritableStream<SubscribeEvidenceRequest, SubscribeEvidenceResponse>) {
      assert.equal(call.request.productRunId, "product-run-1");
      call.write(SubscribeEvidenceResponse.create({
        evidence: ProductEvidenceEnvelope.create({
          publicationId: "publication-9",
          publicationSequence: 9,
          producedUnixMs: 1_789_387_200_000,
          capitalReservoir: CapitalReservoirEvidence.create({
            eventId: "capital-7",
            pipelineRunId: "pipeline-1",
            collectionRunId: "collection-1",
            runType: "RUN_A",
            eventSequence: 7,
            symbol: "AAPL",
            eventType: CapitalReservoirEventType.CAPITAL_RESERVOIR_EVENT_TYPE_OUTFLOW,
            cause: CapitalReservoirCause.CAPITAL_RESERVOIR_CAUSE_HOP_ON,
            stageEmitValueId: "0123456789abcdef01234567",
            triggerEventUnixMs: 1_789_387_199_000,
            executionEventUnixMs: 1_789_387_199_500,
            processedUnixMs: 1_789_387_200_000,
            quantity: 10,
            executionPrice: 9_985,
            signedFlowAmount: -99_850,
            reservoirBefore: 3_000_000,
            reservoirAfter: 2_900_150,
            activeQuantityAfter: 10,
          }),
        }),
      }));
      call.write(SubscribeEvidenceResponse.create({
        evidence: ProductEvidenceEnvelope.create({
          publicationId: "publication-10",
          publicationSequence: 10,
          producedUnixMs: 1_789_387_201_000,
          capitalReservoir: CapitalReservoirEvidence.create({
            eventId: "capital-8",
            pipelineRunId: "pipeline-1",
            collectionRunId: "collection-1",
            runType: "RUN_A",
            eventSequence: 8,
            symbol: "AAPL",
            eventType: CapitalReservoirEventType.CAPITAL_RESERVOIR_EVENT_TYPE_INFLOW,
            cause: CapitalReservoirCause.CAPITAL_RESERVOIR_CAUSE_HOP_OFF,
            processedUnixMs: 1_789_387_201_000,
            quantity: 10,
            executionPrice: 9_990,
            signedFlowAmount: 99_900,
            reservoirBefore: 2_900_150,
            reservoirAfter: 3_000_050,
            activeQuantityAfter: 0,
          }),
        }),
      }));
    },
  });
  const port = await new Promise<number>((resolve, reject) => {
    server.bindAsync("127.0.0.1:0", ServerCredentials.createInsecure(), (error, boundPort) => {
      if (error) reject(error);
      else resolve(boundPort);
    });
  });
  const previousAddress = process.env.JEH_HOHO_SERVER_ADDRESS;
  const previousRun = process.env.JEH_HOHO_PRODUCT_RUN_ID;
  process.env.JEH_HOHO_SERVER_ADDRESS = `127.0.0.1:${port}`;
  process.env.JEH_HOHO_PRODUCT_RUN_ID = "product-run-1";
  let stop: (() => void) | undefined;
  try {
    const received = await new Promise<CapitalReservoirEvent[]>((resolve, reject) => {
      const timeout = setTimeout(() => reject(new Error("timed out waiting for generated Capital event")), 2_000);
      const events: CapitalReservoirEvent[] = [];
      stop = subscribeToReservoirEvents(
        (event) => {
          events.push(event);
          if (events.length === 2) {
            clearTimeout(timeout);
            resolve(events);
          }
        },
        (error) => {
          clearTimeout(timeout);
          reject(error);
        },
      );
    });
    assert.equal(received[0].eventId, "capital-7");
    assert.equal(received[0].eventSequence, 7);
    assert.equal(received[0].eventType, "OUTFLOW");
    assert.equal(received[0].cause, "HOP_ON");
    assert.equal(received[0].signedFlowAmount, -99_850);
    assert.equal(received[0].reservoirAfter, 2_900_150);
    assert.equal(received[1].eventId, "capital-8");
    assert.equal(received[1].eventSequence, 8);
    assert.equal(received[1].eventType, "INFLOW");
    assert.equal(received[1].reservoirAfter, 3_000_050);
  } finally {
    stop?.();
    if (previousAddress === undefined) delete process.env.JEH_HOHO_SERVER_ADDRESS;
    else process.env.JEH_HOHO_SERVER_ADDRESS = previousAddress;
    if (previousRun === undefined) delete process.env.JEH_HOHO_PRODUCT_RUN_ID;
    else process.env.JEH_HOHO_PRODUCT_RUN_ID = previousRun;
    await new Promise<void>((resolve) => server.tryShutdown(() => resolve()));
  }
});