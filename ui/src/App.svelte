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
  import type { Alert, GraphEdge, GraphNode, GraphView, Snapshot, Vehicle } from './types';

  let snapshot = $state<Snapshot | null>(null);
  let selectedID = $state<string>('');
  let commandBusy = $state<string>('');
  let commandError = $state<string>('');
  let graphView = $state<GraphView | null>(null);
  let graphSource = $state<string>('semlink');
  let graphError = $state<string>('');
  let graphLoading = $state(false);
  let lastGraphVehicleID = $state('');

  const vehicles = $derived(snapshot?.vehicles ?? []);
  const alerts = $derived((snapshot?.alerts ?? []).filter((alert) => alert.active).slice(0, 8));
  const selected = $derived(vehicles.find((vehicle) => vehicle.entity_id === selectedID) ?? vehicles[0]);
  const metrics = $derived(snapshot?.metrics);
  const bounds = $derived(makeBounds(vehicles));
  const graphLenses = $derived(graphView?.lenses ?? []);
  const activeGraphLens = $derived(graphLenses.find((lens) => lens.source === graphSource) ?? graphLenses[0]);
  const graphNodes = $derived(layoutGraph(activeGraphLens?.nodes ?? []));
  const graphEdges = $derived(layoutEdges(activeGraphLens?.edges ?? [], graphNodes));

  onMount(() => {
    void fetchSnapshot();
    const events = new EventSource('/api/events');
    events.addEventListener('snapshot', (event) => {
      snapshot = JSON.parse((event as MessageEvent).data) as Snapshot;
      if (!selectedID && snapshot.vehicles.length > 0) {
        selectedID = snapshot.vehicles[0].entity_id;
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
    if (selectedID && selectedID !== lastGraphVehicleID) {
      lastGraphVehicleID = selectedID;
      void fetchGraph(selectedID);
    }
  });

  async function fetchSnapshot() {
    const response = await fetch('/api/snapshot');
    snapshot = (await response.json()) as Snapshot;
    if (!selectedID && snapshot.vehicles.length > 0) {
      selectedID = snapshot.vehicles[0].entity_id;
    }
  }

  async function fetchGraph(vehicleID = selectedID) {
    if (!vehicleID || graphLoading) return;
    graphLoading = true;
    graphError = '';
    try {
      const response = await fetch(`/api/graph?vehicle_id=${encodeURIComponent(vehicleID)}`);
      if (!response.ok) {
        graphError = await response.text();
        return;
      }
      graphView = (await response.json()) as GraphView;
      if (!graphView.lenses.some((lens) => lens.source === graphSource)) {
        graphSource = graphView.lenses[0]?.source ?? 'semlink';
      }
    } catch (error) {
      graphError = error instanceof Error ? error.message : 'graph unavailable';
    } finally {
      graphLoading = false;
    }
  }

  async function sendCommand(verb: string) {
    if (!selected) return;
    commandBusy = verb;
    commandError = '';
    try {
      const response = await fetch('/api/commands', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ vehicle_id: selected.entity_id, verb })
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

  function layoutGraph(nodes: GraphNode[]): PositionedNode[] {
    if (nodes.length === 0) return [];
    const placed: PositionedNode[] = [];
    for (const [index, node] of nodes.entries()) {
      if (index === 0) {
        placed.push({ ...node, x: 50, y: 50 });
        continue;
      }
      const count = Math.max(1, nodes.length - 1);
      const angle = -Math.PI / 2 + ((index - 1) / count) * Math.PI * 2;
      placed.push({
        ...node,
        x: Math.min(84, Math.max(16, 50 + Math.cos(angle) * 35)),
        y: Math.min(82, Math.max(18, 50 + Math.sin(angle) * 31))
      });
    }
    return placed;
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

  function makeBounds(items: Vehicle[]) {
    if (items.length === 0) {
      return { minLat: 38.88, maxLat: 38.9, minLon: -77.05, maxLon: -77.02 };
    }
    const lats = items.map((item) => item.latitude_deg || 38.8895);
    const lons = items.map((item) => item.longitude_deg || -77.0353);
    const minLat = Math.min(...lats);
    const maxLat = Math.max(...lats);
    const minLon = Math.min(...lons);
    const maxLon = Math.max(...lons);
    return {
      minLat: minLat - 0.003,
      maxLat: maxLat + 0.003,
      minLon: minLon - 0.003,
      maxLon: maxLon + 0.003
    };
  }

  function markerStyle(vehicle: Vehicle) {
    const x = ((vehicle.longitude_deg - bounds.minLon) / (bounds.maxLon - bounds.minLon || 1)) * 100;
    const y = 100 - ((vehicle.latitude_deg - bounds.minLat) / (bounds.maxLat - bounds.minLat || 1)) * 100;
    return `left:${Math.min(96, Math.max(4, x))}%;top:${Math.min(94, Math.max(6, y))}%;transform:rotate(${vehicle.heading_deg}deg);`;
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
            class:selected={selected?.entity_id === vehicle.entity_id}
            class:offline={vehicle.link_status === 'lost'}
            onclick={() => (selectedID = vehicle.entity_id)}
            type="button"
          >
            <span class="callsign">{vehicle.callsign}</span>
            <span class="status-dot {linkClass(vehicle)}"></span>
            <span>{vehicle.battery_remaining}%</span>
          </button>
        {/each}
      </div>
    </aside>

    <section class="panel map-panel">
      <div class="panel-title">
        <MapPin size={18} />
        <h2>Mission Map</h2>
      </div>
      <div class="mission-map">
        <div class="runway"></div>
        {#each vehicles as vehicle}
          <button
            class="marker {batteryClass(vehicle)}"
            class:selected={selected?.entity_id === vehicle.entity_id}
            style={markerStyle(vehicle)}
            title={vehicle.callsign}
            aria-label={vehicle.callsign}
            onclick={() => (selectedID = vehicle.entity_id)}
            type="button"
          >
            <span></span>
          </button>
        {/each}
      </div>
    </section>

    <aside class="panel detail">
      {#if selected}
        <div class="panel-title">
          <ShieldCheck size={18} />
          <h2>{selected.callsign}</h2>
        </div>
        <div class="readouts">
          <div>
            <span class="metric-label">Link</span>
            <strong class={linkClass(selected)}>{selected.link_status}</strong>
          </div>
          <div>
            <span class="metric-label">Battery</span>
            <strong class={batteryClass(selected)}>{selected.battery_remaining}%</strong>
          </div>
          <div>
            <span class="metric-label">Altitude</span>
            <strong>{selected.altitude_m.toFixed(1)} m</strong>
          </div>
          <div>
            <span class="metric-label">Speed</span>
            <strong>{selected.ground_speed_mps.toFixed(1)} m/s</strong>
          </div>
          <div>
            <span class="metric-label">Profile</span>
            <strong>{selected.indexing_profile}</strong>
          </div>
          <div>
            <span class="metric-label">Revision</span>
            <strong>{selected.graph_revision || '...'}</strong>
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
        </div>
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
      </div>

      {#if graphError}
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
