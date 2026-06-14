<script lang="ts">
  import { onMount } from 'svelte';
  import {
    AlertTriangle,
    Battery,
    CirclePause,
    Home,
    MapPin,
    Radio,
    Send,
    ShieldCheck
  } from '@lucide/svelte';
  import type { Alert, Snapshot, Vehicle } from './types';

  let snapshot = $state<Snapshot | null>(null);
  let selectedID = $state<string>('');
  let commandBusy = $state<string>('');
  let commandError = $state<string>('');

  const vehicles = $derived(snapshot?.vehicles ?? []);
  const alerts = $derived((snapshot?.alerts ?? []).filter((alert) => alert.active).slice(0, 8));
  const selected = $derived(vehicles.find((vehicle) => vehicle.entity_id === selectedID) ?? vehicles[0]);
  const metrics = $derived(snapshot?.metrics);
  const bounds = $derived(makeBounds(vehicles));

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
    return () => events.close();
  });

  async function fetchSnapshot() {
    const response = await fetch('/api/snapshot');
    snapshot = (await response.json()) as Snapshot;
    if (!selectedID && snapshot.vehicles.length > 0) {
      selectedID = snapshot.vehicles[0].entity_id;
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
  </section>
</main>
