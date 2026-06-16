export type Vehicle = {
  entity_id: string;
  callsign: string;
  system_id: number;
  armed: boolean;
  mode: string;
  flight_status: string;
  link_status: string;
  battery_remaining: number;
  voltage_battery_mv: number;
  latitude_deg: number;
  longitude_deg: number;
  altitude_m: number;
  ground_speed_mps: number;
  heading_deg: number;
  sequence: number;
  last_seen: string;
  graph_revision: number;
  indexing_profile: string;
};

export type Alert = {
  entity_id: string;
  kind: string;
  severity: string;
  active: boolean;
  subject_entity: string;
  message: string;
  raised_at: string;
  graph_revision: number;
};

export type Command = {
  entity_id: string;
  target_entity: string;
  verb: string;
  status: string;
  requested_at: string;
  graph_revision: number;
};

export type COPKind = 'operator' | 'marker' | 'message';

export type COPView = {
  kind: COPKind;
  entity_id: string;
  uid: string;
  callsign?: string;
  label?: string;
  description?: string;
  text?: string;
  sender_uid?: string;
  sender_entity?: string;
  latitude_deg?: number;
  longitude_deg?: number;
  altitude_m?: number;
  heading_deg?: number;
  ground_speed_mps?: number;
  has_position: boolean;
  last_seen: string;
  graph_revision: number;
  indexing_profile: string;
};

export type Metrics = {
  raw_frames: number;
  decoded_frames: number;
  decode_errors: number;
  projected_writes: number;
  graph_writes: number;
  graph_errors: number;
  raw_publish_errors: number;
  buffer_drops: number;
  buffer_size: number;
  buffer_capacity: number;
  frames_per_second: number;
  graph_writes_per_second: number;
  last_graph_latency_ms: number;
  nats_url: string;
  semstreams_embedded: boolean;
  started_at: string;
};

export type Snapshot = {
  generated_at: string;
  vehicles: Vehicle[];
  alerts: Alert[];
  commands: Command[];
  operators: COPView[];
  markers: COPView[];
  messages: COPView[];
  metrics: Metrics;
};

export type GraphNode = {
  id: string;
  label: string;
  kind: string;
  profile?: string;
  detail?: string;
  status?: string;
};

export type GraphEdge = {
  from: string;
  to: string;
  label: string;
};

export type GraphFact = {
  subject: string;
  predicate: string;
  object: string;
  source?: string;
};

export type GraphStat = {
  label: string;
  value: string;
};

export type GraphLens = {
  source: 'semlink' | 'csapi' | string;
  label: string;
  status: string;
  summary: string;
  nodes: GraphNode[] | null;
  edges: GraphEdge[] | null;
  facts: GraphFact[] | null;
  stats: GraphStat[] | null;
};

export type GraphView = {
  generated_at: string;
  entity_id: string;
  entity_kind: string;
  entity_label: string;
  vehicle_id?: string;
  lenses: GraphLens[];
};
