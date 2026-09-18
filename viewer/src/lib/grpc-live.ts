// JEH_HOHO product live bridge; replaces the reference viewer's runtime proto loading.
import "server-only";

import { credentials } from "@grpc/grpc-js";
import {
  CapitalReservoirEvidence,
  ProductEvidenceServiceClient,
} from "@/generated/jeh_hoho/v1/JEH_HOHO";
import { normalizeReservoirEvent, type CapitalReservoirEvent } from "./capital-reservoir";

export function subscribeToReservoirEvents(
  onEvent: (event: CapitalReservoirEvent) => void,
  onError: (error: Error) => void,
): () => void {
  const address = process.env.JEH_HOHO_SERVER_ADDRESS?.trim() || "127.0.0.1:50052";
  const client = new ProductEvidenceServiceClient(address, credentials.createInsecure());
  const stream = client.subscribeEvidence({
    productRunId: process.env.JEH_HOHO_PRODUCT_RUN_ID?.trim() || "",
  });
  stream.on("data", (response) => {
    const evidence = response.evidence?.capitalReservoir;
    if (!evidence) return;
    try {
      onEvent(normalizeReservoirEvent(CapitalReservoirEvidence.toJSON(evidence)));
    } catch (error) {
      onError(error instanceof Error ? error : new Error("Invalid live Capital Reservoir event"));
    }
  });
  stream.on("error", onError);
  return () => {
    stream.cancel();
    client.close();
  };
}