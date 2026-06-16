<script lang="ts">
  import { onMount } from 'svelte';
  import {
    AlertTriangle,
    Battery,
    CirclePause,
    Database,
    GitFork,
    Home,
    MapPin,
    Radio,
    Send,
    ShieldCheck
  } from '@lucide/svelte';
  import type { Alert, COPKind, COPView, GraphEdge, GraphNode, GraphView, Snapshot, Vehicle } from './types';

  let snapshot = $state<Snapshot | null>(null);
  let selectedID = $state<string>('');
  let commandBusy = $state<string>('');
  let commandError = $state<string>('');
  let graphView = $state<GraphView | null>(null);
  let graphSource = $state<string>('semlink');
  let graphError = $state<string>('');
  let graphLoading = $state(false);
  let graphLoadingEntityID = $state('');
  let graphReplacing = $state(false);
  let lastGraphEntityID = $state('');
  let graphRequestSeq = 0;

  const vehicles = $derived(snapshot?.vehicles ?? []);
  const operators = $derived(snapshot?.operators ?? []);
  const markers = $derived(snapshot?.markers ?? []);
  const messages = $derived((snapshot?.messages ?? []).filter((message) => message.has_position));
  const copItems = $derived([...operators, ...markers, ...messages]);
  const alerts = $derived((snapshot?.alerts ?? []).filter((alert) => alert.active).slice(0, 8));
  const selectedVehicle = $derived(vehicles.find((vehicle) => vehicle.entity_id === selectedID));
  const selectedCOP = $derived(copItems.find((item) => item.entity_id === selectedID));
  const selectedTitle = $derived(selectedVehicle?.callsign ?? (selectedCOP ? copTitle(selectedCOP) : ''));
  const metrics = $derived(snapshot?.metrics);
  const graphLenses = $derived(graphView?.lenses ?? []);
  const activeGraphLens = $derived(graphLenses.find((lens) => lens.source === graphSource) ?? graphLenses[0]);
  const graphNodes = $derived(layoutGraph(activeGraphLens?.nodes ?? [], activeGraphLens?.edges ?? []));
  const graphEdges = $derived(layoutEdges(activeGraphLens?.edges ?? [], graphNodes));

  onMount(() => {
    void fetchSnapshot();
    const events = new EventSource('/api/events');
    events.addEventListener('snapshot', (event) => {
      snapshot = JSON.parse((event as MessageEvent).data) as Snapshot;
      if (!selectedID && snapshot.vehicles.length > 0) {
        selectEntity(snapshot.vehicles[0].entity_id);
      }
    });
    events.onerror = () => {
      events.close();
      setTimeout(() => void fetchSnapshot(), 1000);
    };
    const graphTimer = window.setInterval(() => void fetchGraph(), 3000);
    return () => {
      events.close();
      window.clearInterval(graphTimer);
    };
  });

  $effect(() => {
    if (selectedID && selectedID !== lastGraphEntityID) {
      lastGraphEntityID = selectedID;
      void fetchGraph(selectedID);
    }
  });

  async function fetchSnapshot() {
    const response = await fetch('/api/snapshot');
    snapshot = (await response.json()) as Snapshot;
    if (!selectedID && snapshot.vehicles.length > 0) {
      selectEntity(snapshot.vehicles[0].entity_id);
    }
  }

  function selectEntity(entityID: string) {
    selectedID = entityID;
    if (graphView?.entity_id !== entityID) {
      graphError = '';
      graphReplacing = true;
      graphLoadingEntityID = entityID;
    }
  }

  async function fetchGraph(entityID = selectedID) {
    if (!entityID) return;
    if (graphLoading && graphLoadingEntityID === entityID) return;
    const requestSeq = ++graphRequestSeq;
    const entityChanged = graphView?.entity_id !== entityID;
    graphLoading = true;
    graphLoadingEntityID = entityID;
    graphReplacing = entityChanged;
    graphError = '';
    if (entityChanged) {
      graphView = null;
    }
    try {
      const response = await fetch(`/api/graph?entity_id=${encodeURIComponent(entityID)}`);
      if (requestSeq !== graphRequestSeq || entityID !== selectedID) return;
      if (!response.ok) {
        graphError = await response.text();
        return;
      }
      graphView = (await response.json()) as GraphView;
      if (!graphView.lenses.some((lens) => lens.source === graphSource)) {
        graphSource = graphView.lenses[0]?.source ?? 'semlink';
      }
    } catch (error) {
      if (requestSeq === graphRequestSeq) {
        graphError = error instanceof Error ? error.message : 'graph unavailable';
      }
    } finally {
      if (requestSeq === graphRequestSeq) {
        graphLoading = false;
        graphLoadingEntityID = '';
        graphReplacing = false;
      }
    }
  }

  async function sendCommand(verb: string) {
    if (!selectedVehicle) return;
    commandBusy = verb;
    commandError = '';
    try {
      const response = await fetch('/api/commands', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ vehicle_id: selectedVehicle.entity_id, verb })
      });
      if (!response.ok) {
        commandError = await response.text();
      }
    } finally {
      commandBusy = '';
    }
  }

  type PositionedNode = GraphNode & { x: number; y: number };
  type PositionedEdge = GraphEdge & { x1: number; y1: number; x2: number; y2: number };

  function layoutGraph(nodes: GraphNode[], edges: GraphEdge[]): PositionedNode[] {
    if (nodes.length === 0) return [];

    const nodeIDs = new Set(nodes.map((node) => node.id));
    const validEdges = edges.filter((edge) => nodeIDs.has(edge.from) && nodeIDs.has(edge.to) && edge.from !== edge.to);
    const incomingIDs = new Set(validEdges.map((edge) => edge.to));
    const depths = new Map<string, number>();

    for (const [index, node] of nodes.entries()) {
      depths.set(node.id, index === 0 || !incomingIDs.has(node.id) ? 0 : 1);
    }

    const maxDepth = Math.min(3, Math.max(1, nodes.length - 1));
    for (let pass = 0; pass < nodes.length; pass += 1) {
      let changed = false;
      for (const edge of validEdges) {
        const nextDepth = Math.min(maxDepth, (depths.get(edge.from) ?? 0) + 1);
        if (nextDepth > (depths.get(edge.to) ?? 0)) {
          depths.set(edge.to, nextDepth);
          changed = true;
        }
      }
      if (!changed) break;
    }

    const groups = new Map<number, GraphNode[]>();
    for (const node of nodes) {
      const depth = depths.get(node.id) ?? 0;
      groups.set(depth, [...(groups.get(depth) ?? []), node]);
    }

    const usedDepths = [...groups.keys()].sort((a, b) => a - b);
    const rightmostDepth = Math.max(...usedDepths, 0);
    return nodes.map((node) => {
      const depth = depths.get(node.id) ?? 0;
      const layer = groups.get(depth) ?? [node];
      const layerIndex = layer.findIndex((item) => item.id === node.id);
      const x = rightmostDepth === 0 ? 50 : 20 + (60 * depth) / rightmostDepth;
      return {
        ...node,
        x,
        y: graphLayerY(layerIndex, layer.length)
      };
    });
  }

  function graphLayerY(index: number, count: number) {
    if (count <= 1) return 50;
    if (count === 2) return index === 0 ? 34 : 66;
    return 18 + (64 * index) / (count - 1);
  }

  function layoutEdges(edges: GraphEdge[], nodes: PositionedNode[]): PositionedEdge[] {
    return edges
      .map((edge) => {
        const from = nodes.find((node) => node.id === edge.from);
        const to = nodes.find((node) => node.id === edge.to);
        if (!from || !to) return null;
        return { ...edge, x1: from.x, y1: from.y, x2: to.x, y2: to.y };
      })
      .filter((edge): edge is PositionedEdge => edge !== null);
  }

  function graphNodeStyle(node: PositionedNode) {
    return `left:${node.x}%;top:${node.y}%;`;
  }

  function graphNodeClass(node: GraphNode) {
    const status = (node.status ?? '').toLowerCase();
    if (status.includes('critical') || status.includes('lost')) return 'danger-node';
    if (status.includes('warning') || status.includes('requested')) return 'warn-node';
    if (node.profile === 'control' || node.kind.toLowerCase().includes('command')) return 'control-node';
    return 'signal-node';
  }

  type MapPoint = {
    latitude_deg?: number;
    longitude_deg?: number;
  };

  type Bounds = {
    minLat: number;
    maxLat: number;
    minLon: number;
    maxLon: number;
  };

  const bounds: Bounds = { minLat: 38.875, maxLat: 38.91, minLon: -77.065, maxLon: -77.018 };

  function markerStyle(vehicle: Vehicle) {
    return `${pointStyle(vehicle)}transform:rotate(${vehicle.heading_deg}deg);`;
  }

  function pointStyle(point: MapPoint) {
    const x = (((point.longitude_deg ?? -77.0353) - bounds.minLon) / (bounds.maxLon - bounds.minLon || 1)) * 100;
    const y = 100 - (((point.latitude_deg ?? 38.8895) - bounds.minLat) / (bounds.maxLat - bounds.minLat || 1)) * 100;
    return `left:${Math.min(96, Math.max(4, x))}%;top:${Math.min(94, Math.max(6, y))}%;`;
  }

  function messagePoint(item: COPView): MapPoint {
    const sender = operators.find(
      (operator) =>
        operator.has_position &&
        (operator.entity_id === item.sender_entity || (!!item.sender_uid && operator.uid === item.sender_uid))
    );
    return sender ?? item;
  }

  function messageStyle(item: COPView) {
    return `${pointStyle(messagePoint(item))}transform:translate(13px, -15px);`;
  }

  function messageLabelStyle(item: COPView) {
    return `${pointStyle(messagePoint(item))}transform:translate(22px, -34px);`;
  }

  function copTitle(item: COPView) {
    if (item.kind === 'message') return item.text ?? item.callsign ?? item.sender_uid ?? item.uid;
    if (item.kind === 'operator') return item.callsign ?? item.uid;
    return item.label ?? item.description ?? item.uid;
  }

  function messageTitle(item: COPView) {
    const from = item.callsign || item.sender_uid || item.uid;
    return item.text ? `${from}: ${item.text}` : from;
  }

  function markerAriaLabel(vehicle: Vehicle) {
    return `${vehicle.callsign} UAV`;
  }

  function copAriaLabel(item: COPView) {
    const label = copTitle(item);
    return `${label} ${item.kind}`;
  }

  function messageAriaLabel(item: COPView) {
    return `${messageTitle(item)} chat`;
  }

  function copKindLabel(kind: COPKind) {
    if (kind === 'operator') return 'Operator';
    if (kind === 'marker') return 'Marker';
    return 'GeoChat';
  }

  function copDetailText(item: COPView) {
    return item.text ?? item.description ?? item.callsign ?? item.label ?? item.uid;
  }

  function copPositionText(item: COPView) {
    if (!item.has_position) return 'none';
    return `${(item.latitude_deg ?? 0).toFixed(5)}, ${(item.longitude_deg ?? 0).toFixed(5)}`;
  }

  function lastSeenText(value: string) {
    if (!value) return 'unknown';
    const date = new Date(value);
    if (Number.isNaN(date.getTime())) return value;
    return date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit' });
  }

  function fallbackText(value: string | undefined, fallback: string) {
    return value && value.trim() ? value : fallback;
  }

  function shortText(value: string, max = 26) {
    if (value.length <= max) return value;
    return `${value.slice(0, max - 3)}...`;
  }

  function batteryClass(vehicle: Vehicle) {
    if (vehicle.battery_remaining <= 25) return 'danger';
    if (vehicle.battery_remaining <= 45) return 'warn';
    return 'ok';
  }

  function linkClass(vehicle: Vehicle) {
    return vehicle.link_status === 'lost' ? 'danger' : 'ok';
  }

  function alertClass(alert: Alert) {
    return alert.severity === 'critical' ? 'danger' : 'warn';
  }
</script>

<main class="shell">
  <header class="topbar">
    <div>
      <h1>SemGCS</h1>
      <p>{vehicles.length} vehicles · {metrics?.nats_url ?? 'connecting'}</p>
    </div>
    <div class="top-metrics">
      <div>
        <span>{metrics?.frames_per_second.toFixed(1) ?? '0.0'}</span>
        <span class="metric-label">frames/s</span>
      </div>
      <div>
        <span>{metrics?.graph_writes_per_second.toFixed(1) ?? '0.0'}</span>
        <span class="metric-label">graph/s</span>
      </div>
      <div>
        <span>{metrics?.buffer_drops ?? 0}</span>
        <span class="metric-label">drops</span>
      </div>
    </div>
  </header>

  <section class="dashboard">
    <aside class="panel fleet">
      <div class="panel-title">
        <Radio size={18} />
        <h2>Fleet</h2>
      </div>
      <div class="vehicle-list">
        {#each vehicles as vehicle}
          <button
            class:selected={selectedVehicle?.entity_id === vehicle.entity_id}
            class:offline={vehicle.link_status === 'lost'}
            onclick={() => selectEntity(vehicle.entity_id)}
            type="button"
          >
            <span class="callsign">{vehicle.callsign}</span>
            <span class="status-dot {linkClass(vehicle)}"></span>
            <span>{vehicle.battery_remaining}%</span>
          </button>
        {/each}
      </div>

      {#if copItems.length > 0}
        <div class="cop-sidebar">
          <div class="sidebar-kicker">TAK COP</div>
          {#if operators.length > 0}
            <div class="cop-sidebar-group">
              <span>Operators</span>
              {#each operators as operator}
                <button
                  class:selected={selectedCOP?.entity_id === operator.entity_id}
                  onclick={() => selectEntity(operator.entity_id)}
                  type="button"
                >
                  <i class="cop-swatch operator"></i>
                  <span>{shortText(fallbackText(operator.callsign, operator.uid), 18)}</span>
                  <small>{operator.graph_revision || '...'}</small>
                </button>
              {/each}
            </div>
          {/if}
          {#if markers.length > 0}
            <div class="cop-sidebar-group">
              <span>Markers</span>
              {#each markers as marker}
                <button
                  class:selected={selectedCOP?.entity_id === marker.entity_id}
                  onclick={() => selectEntity(marker.entity_id)}
                  type="button"
                >
                  <i class="cop-swatch marker-poi"></i>
                  <span>{shortText(fallbackText(marker.label, marker.uid), 18)}</span>
                  <small>{marker.graph_revision || '...'}</small>
                </button>
              {/each}
            </div>
          {/if}
          {#if messages.length > 0}
            <div class="cop-sidebar-group">
              <span>GeoChat</span>
              {#each messages as message}
                <button
                  class:selected={selectedCOP?.entity_id === message.entity_id}
                  onclick={() => selectEntity(message.entity_id)}
                  type="button"
                >
                  <i class="cop-swatch message"></i>
                  <span>{shortText(messageTitle(message), 24)}</span>
                  <small>{message.graph_revision || '...'}</small>
                </button>
              {/each}
            </div>
          {/if}
        </div>
      {/if}
    </aside>

    <section class="panel map-panel">
      <div class="panel-title">
        <MapPin size={18} />
        <h2>Mission Map</h2>
      </div>
      <div class="mission-map">
        <div class="runway"></div>
        <div class="map-legend">
          <span><i class="legend-swatch uav"></i>UAV</span>
          <span><i class="legend-swatch operator"></i>Operator</span>
          <span><i class="legend-swatch marker-poi"></i>Marker</span>
          <span><i class="legend-swatch message"></i>Chat</span>
        </div>
        {#each vehicles as vehicle}
	          <button
	            class="marker {batteryClass(vehicle)}"
	            class:selected={selectedVehicle?.entity_id === vehicle.entity_id}
	            style={markerStyle(vehicle)}
	            title={vehicle.callsign}
	            aria-label={markerAriaLabel(vehicle)}
            onclick={() => selectEntity(vehicle.entity_id)}
            type="button"
          >
            <span></span>
          </button>
	        {/each}
	        {#each operators.filter((item) => item.has_position) as operator}
	          <button
	            class="cop-dot operator-dot"
	            class:selected={selectedCOP?.entity_id === operator.entity_id}
	            style={pointStyle(operator)}
	            title={copTitle(operator)}
	            aria-label={copAriaLabel(operator)}
	            onclick={() => selectEntity(operator.entity_id)}
	            type="button"
	          ></button>
	          <span class="map-label operator-label" style={pointStyle(operator)}>{shortText(fallbackText(operator.callsign, operator.uid), 18)}</span>
	        {/each}
	        {#each markers.filter((item) => item.has_position) as marker}
	          <button
	            class="cop-dot poi-dot"
	            class:selected={selectedCOP?.entity_id === marker.entity_id}
	            style={pointStyle(marker)}
	            title={copTitle(marker)}
	            aria-label={copAriaLabel(marker)}
	            onclick={() => selectEntity(marker.entity_id)}
	            type="button"
	          ></button>
	          <span class="map-label poi-label" style={pointStyle(marker)}>{shortText(fallbackText(marker.label, marker.uid), 18)}</span>
	        {/each}
	        {#each messages as message}
	          <button
	            class="cop-dot message-dot"
	            class:selected={selectedCOP?.entity_id === message.entity_id}
	            style={messageStyle(message)}
	            title={messageTitle(message)}
	            aria-label={messageAriaLabel(message)}
	            onclick={() => selectEntity(message.entity_id)}
	            type="button"
	          ></button>
	          <span class="map-label message-label" style={messageLabelStyle(message)}>{shortText(fallbackText(message.text, message.uid), 22)}</span>
	        {/each}
	      </div>
	    </section>

	    <aside class="panel detail">
	      {#if selectedVehicle}
	        <div class="panel-title">
	          <ShieldCheck size={18} />
	          <h2>{selectedVehicle.callsign}</h2>
	        </div>
	        <div class="readouts">
	          <div>
	            <span class="metric-label">Link</span>
	            <strong class={linkClass(selectedVehicle)}>{selectedVehicle.link_status}</strong>
	          </div>
	          <div>
	            <span class="metric-label">Battery</span>
	            <strong class={batteryClass(selectedVehicle)}>{selectedVehicle.battery_remaining}%</strong>
	          </div>
	          <div>
	            <span class="metric-label">Altitude</span>
	            <strong>{selectedVehicle.altitude_m.toFixed(1)} m</strong>
	          </div>
	          <div>
	            <span class="metric-label">Speed</span>
	            <strong>{selectedVehicle.ground_speed_mps.toFixed(1)} m/s</strong>
	          </div>
	          <div>
	            <span class="metric-label">Profile</span>
	            <strong>{selectedVehicle.indexing_profile}</strong>
	          </div>
	          <div>
	            <span class="metric-label">Revision</span>
	            <strong>{selectedVehicle.graph_revision || '...'}</strong>
	          </div>
	        </div>
        <div class="commands">
          <button aria-label="Return to launch" onclick={() => sendCommand('return-to-launch')} disabled={commandBusy !== ''} type="button">
            <Home size={16} />
            <span>Return</span>
          </button>
          <button aria-label="Hold position" onclick={() => sendCommand('hold-position')} disabled={commandBusy !== ''} type="button">
            <CirclePause size={16} />
            <span>Hold</span>
          </button>
          <button aria-label="Land" onclick={() => sendCommand('land')} disabled={commandBusy !== ''} type="button">
            <Send size={16} />
            <span>Land</span>
          </button>
        </div>
	        {#if commandError}
	          <p class="error-line">{commandError}</p>
	        {/if}
	      {:else if selectedCOP}
	        <div class="panel-title">
	          <MapPin size={18} />
	          <h2>{copTitle(selectedCOP)}</h2>
	        </div>
	        <div class="readouts">
	          <div>
	            <span class="metric-label">Kind</span>
	            <strong>{copKindLabel(selectedCOP.kind)}</strong>
	          </div>
	          <div>
	            <span class="metric-label">Profile</span>
	            <strong>{selectedCOP.indexing_profile}</strong>
	          </div>
	          <div>
	            <span class="metric-label">Position</span>
	            <strong>{copPositionText(selectedCOP)}</strong>
	          </div>
	          <div>
	            <span class="metric-label">Last Seen</span>
	            <strong>{lastSeenText(selectedCOP.last_seen)}</strong>
	          </div>
	          <div>
	            <span class="metric-label">Revision</span>
	            <strong>{selectedCOP.graph_revision || '...'}</strong>
	          </div>
	          <div>
	            <span class="metric-label">UID</span>
	            <strong>{shortText(selectedCOP.uid, 22)}</strong>
	          </div>
	        </div>
	        <p class="cop-detail-text">{copDetailText(selectedCOP)}</p>
	        {#if selectedCOP.kind === 'message' && selectedCOP.sender_uid}
	          <p class="quiet">from {selectedCOP.sender_uid}</p>
	        {/if}
	      {/if}
	    </aside>

    <section class="panel telemetry">
      <div class="panel-title">
        <Battery size={18} />
        <h2>Telemetry</h2>
      </div>
      <div class="metrics-grid">
        <div><span class="metric-label">Raw</span><strong>{metrics?.raw_frames ?? 0}</strong></div>
        <div><span class="metric-label">Decoded</span><strong>{metrics?.decoded_frames ?? 0}</strong></div>
        <div><span class="metric-label">Projected</span><strong>{metrics?.projected_writes ?? 0}</strong></div>
        <div><span class="metric-label">Graph</span><strong>{metrics?.graph_writes ?? 0}</strong></div>
        <div><span class="metric-label">Buffer</span><strong>{metrics?.buffer_size ?? 0}/{metrics?.buffer_capacity ?? 0}</strong></div>
        <div><span class="metric-label">Latency</span><strong>{metrics?.last_graph_latency_ms.toFixed(1) ?? '0.0'} ms</strong></div>
      </div>
    </section>

    <section class="panel alerts">
      <div class="panel-title">
        <AlertTriangle size={18} />
        <h2>Alerts</h2>
      </div>
      <div class="alert-list">
        {#each alerts as alert}
          <article class="alert-row {alertClass(alert)}">
            <strong>{alert.kind}</strong>
            <span>{alert.message}</span>
            <small>rev {alert.graph_revision || '...'}</small>
          </article>
        {:else}
          <p class="quiet">Clear</p>
        {/each}
      </div>
    </section>

    <section class="panel graph-panel">
	      <div class="panel-title graph-title">
	        <div class="title-left">
	          <GitFork size={18} />
	          <h2>Graph</h2>
	          {#if selectedTitle}
	            <small>{selectedTitle}</small>
	          {/if}
	        </div>
        {#if graphReplacing && graphLoadingEntityID === selectedID}
          <div class="source-tabs loading-tabs" aria-live="polite">
            <span class="graph-loading-chip"><i class="spinner"></i> Loading lenses</span>
          </div>
        {:else}
          <div class="source-tabs" role="tablist" aria-label="Graph source">
            {#each graphLenses as lens}
              <button
                class:active={activeGraphLens?.source === lens.source}
                onclick={() => (graphSource = lens.source)}
                type="button"
                title={lens.summary}
              >
                {#if lens.source === 'csapi'}
                  <Database size={15} />
                {:else}
                  <GitFork size={15} />
                {/if}
                <span>{lens.label}</span>
                <small>{lens.status}</small>
              </button>
            {/each}
          </div>
        {/if}
      </div>

      {#if graphReplacing && graphLoadingEntityID === selectedID}
        <div class="graph-loading-state" aria-live="polite">
          <span class="spinner large"></span>
          <strong>Loading graph</strong>
          <small>{selectedTitle}</small>
        </div>
      {:else if graphError}
        <p class="error-line">{graphError}</p>
      {:else if activeGraphLens}
        <div class="graph-meta">
          {#each activeGraphLens.stats ?? [] as stat}
            <div>
              <span class="metric-label">{stat.label}</span>
              <strong>{stat.value}</strong>
            </div>
          {/each}
        </div>

        {#if graphNodes.length > 0}
          <div class="graph-layout">
            <div class="graph-stage" aria-label={`${activeGraphLens.label} node graph`}>
              <svg class="graph-links" viewBox="0 0 100 100" preserveAspectRatio="none" aria-hidden="true">
                {#each graphEdges as edge}
                  <line x1={edge.x1} y1={edge.y1} x2={edge.x2} y2={edge.y2}></line>
                {/each}
              </svg>
              {#each graphNodes as node}
                <div class="graph-node {graphNodeClass(node)}" style={graphNodeStyle(node)} title={node.id}>
                  <strong>{node.label}</strong>
                  <span>{node.kind}</span>
                  {#if node.detail}
                    <small>{node.detail}</small>
                  {/if}
                </div>
              {/each}
            </div>

            <div class="fact-list">
              {#each activeGraphLens.facts ?? [] as fact}
                <div class="fact-row">
                  <span>{fact.subject}</span>
                  <strong>{fact.predicate}</strong>
                  <span>{fact.object}</span>
                </div>
              {:else}
                <p class="quiet">{activeGraphLens.summary}</p>
              {/each}
            </div>
          </div>
        {:else}
          <p class="quiet">{activeGraphLens.summary}</p>
        {/if}
      {:else}
        <p class="quiet">Waiting for graph</p>
      {/if}
    </section>
  </section>
</main>
