import { fixtureSnapshot } from './fixture';
import type { AssociationReview, AssociationReviewDecision, RuntimeSnapshot, ScenarioStatus, Snapshot } from './types';

export type SnapshotLoadResult = {
  snapshot: Snapshot;
  source: 'api' | 'fixture';
  error?: string;
};

export type RuntimeLoadResult = {
  runtime: RuntimeSnapshot | null;
  error?: string;
};

export type ScenarioStatusLoadResult = {
  status: ScenarioStatus | null;
  error?: string;
};

export type AssociationReviewRequest = {
  decision: AssociationReviewDecision;
  reviewed_by?: string;
  comment?: string;
};

export type AssociationReviewOptions = {
  operatorID?: string;
  fetcher?: typeof fetch;
};

export async function loadSnapshot(fetcher: typeof fetch = fetch): Promise<SnapshotLoadResult> {
  try {
    const response = await fetcher('/api/cop/snapshot', {
      headers: { accept: 'application/json' }
    });
    if (!response.ok) {
      throw new Error(`snapshot request failed: ${response.status}`);
    }
    const snapshot = (await response.json()) as Snapshot;
    return { snapshot, source: 'api' };
  } catch (error) {
    return {
      snapshot: fixtureSnapshot,
      source: 'fixture',
      error: error instanceof Error ? error.message : 'snapshot unavailable'
    };
  }
}

export async function loadRuntime(fetcher: typeof fetch = fetch): Promise<RuntimeLoadResult> {
  try {
    const response = await fetcher('/api/cop/runtime', {
      headers: { accept: 'application/json' }
    });
    if (!response.ok) {
      throw new Error(`runtime request failed: ${response.status}`);
    }
    const runtime = (await response.json()) as RuntimeSnapshot;
    return { runtime };
  } catch (error) {
    return {
      runtime: null,
      error: error instanceof Error ? error.message : 'runtime unavailable'
    };
  }
}

export async function loadScenarioStatus(fetcher: typeof fetch = fetch): Promise<ScenarioStatusLoadResult> {
  try {
    const response = await fetcher('/scenario/status', {
      headers: { accept: 'application/json' }
    });
    if (!response.ok) {
      throw new Error(`scenario status request failed: ${response.status}`);
    }
    const status = (await response.json()) as ScenarioStatus;
    return { status };
  } catch (error) {
    return {
      status: null,
      error: error instanceof Error ? error.message : 'scenario status unavailable'
    };
  }
}

export async function reviewAssociation(
  associationID: string,
  request: AssociationReviewRequest,
  options: AssociationReviewOptions = {}
): Promise<AssociationReview> {
  const fetcher = options.fetcher ?? fetch;
  const headers: Record<string, string> = {
    accept: 'application/json',
    'content-type': 'application/json'
  };
  const operatorID = options.operatorID?.trim();
  if (operatorID) {
    headers['X-SemOps-Operator-ID'] = operatorID;
  }
  const response = await fetcher(`/api/cop/associations/${encodeURIComponent(associationID)}/review`, {
    method: 'POST',
    headers,
    body: JSON.stringify(request)
  });
  if (!response.ok) {
    throw new Error(`association review request failed: ${response.status}`);
  }
  return (await response.json()) as AssociationReview;
}

export function freshnessLabel(isoTime: string, now = new Date()): string {
  const time = Date.parse(isoTime);
  if (Number.isNaN(time)) {
    return 'unknown';
  }
  const ageSeconds = Math.max(0, Math.round((now.getTime() - time) / 1000));
  if (ageSeconds < 60) {
    return `${ageSeconds}s`;
  }
  const ageMinutes = Math.round(ageSeconds / 60);
  if (ageMinutes < 60) {
    return `${ageMinutes}m`;
  }
  return `${Math.round(ageMinutes / 60)}h`;
}

export function formatRate(value: number): string {
  if (!Number.isFinite(value) || value <= 0) {
    return '0';
  }
  if (value < 10) {
    return value.toFixed(1).replace(/\.0$/, '');
  }
  return Math.round(value).toString();
}
