# KLV Packet Storage Materialization Review

Date: 2026-06-22
Scope: KLV decoder worker storage-reference packet materialization.

## Decision

The MISB ST 0601 parser core remains a bounded-byte decoder. Storage-reference-only `semops.klv_packet.v1` payloads
may be decoded only by the decoder component and only when a `PacketMaterializer` is configured. The materializer is
called with `max_packet_bytes`, returns packet bytes, and supplies cleanup for any staged resource.

This keeps object storage, SemSource sidecars, or future media services outside the parser while letting the
SemStreams component flow carry packet references instead of inline bytes when needed.

## Adversarial Findings

- Direct parser support for arbitrary storage refs would hide I/O in what should remain deterministic packet parsing.
- Component-level materialization preserves the flow contract: packet input subject -> bounded materialization ->
  decoded-frame output subject, with no graph writes.
- The decoder rejects storage-ref packets when no materializer is configured, when declared `byte_length` is already
  over `max_packet_bytes`, or when the materializer returns too many bytes.
- The materializer interface should stay small until a real storage backend or SemSource handoff proves whether bytes,
  paths, byte ranges, or streaming reads are the right production shape.

## Evidence

- `go test ./internal/components/klv`

## Follow-Ups

- Add a concrete storage backend adapter only when SemSource or object storage integration is selected.
- Keep public sample and live media claims gated separately from this storage-reference unit path.
