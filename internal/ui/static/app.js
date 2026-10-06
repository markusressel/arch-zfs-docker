let allPackages = [];
let eventSource = null;

document.addEventListener("DOMContentLoaded", () => {
  fetchPackages();
  fetchBuilds();

  // Search filter
  document.getElementById("search-input").addEventListener("input", (e) => {
    filterPackages(e.target.value);
  });

  // Form submit
  document.getElementById("build-form").addEventListener("submit", async (e) => {
    e.preventDefault();
    await triggerBuild();
  });

  // Polling for builds every 8 seconds
  setInterval(fetchBuilds, 8000);
});

async function fetchPackages() {
  try {
    const res = await fetch("/api/packages");
    const data = await res.json();
    allPackages = data.packages || [];

    document.getElementById("stat-count").textContent = data.packageCount || 0;
    document.getElementById("stat-size").textContent = data.totalSizeHuman || "0 B";

    if (data.dbLastModified) {
      const dt = new Date(data.dbLastModified);
      document.getElementById("stat-updated").textContent = dt.toLocaleString();
    } else {
      document.getElementById("stat-updated").textContent = "Not created yet";
    }

    // Determine target kernel from packages
    const linuxPkg = allPackages.find(p => p.packageName === "zfs-linux" && p.kernelVersion);
    if (linuxPkg) {
      document.getElementById("stat-kernel").textContent = linuxPkg.kernelVersion;
    } else {
      document.getElementById("stat-kernel").textContent = "Auto (Latest)";
    }

    renderPackages(allPackages);
  } catch (err) {
    console.error("Failed to fetch packages:", err);
  }
}

function renderPackages(packages) {
  const tbody = document.getElementById("packages-table");
  if (!packages || packages.length === 0) {
    tbody.innerHTML = `<tr><td colspan="6" style="text-align: center; color: var(--text-muted);">No packages found</td></tr>`;
    return;
  }

  tbody.innerHTML = packages.map(p => {
    const dt = new Date(p.modTime).toLocaleDateString();
    return `
      <tr>
        <td><strong>${escapeHtml(p.packageName)}</strong></td>
        <td><code>${escapeHtml(p.version)}</code></td>
        <td><span style="color: var(--accent-blue);">${escapeHtml(p.kernelVersion || "-")}</span></td>
        <td>${escapeHtml(p.sizeHuman)}</td>
        <td style="color: var(--text-secondary);">${dt}</td>
        <td>
          <a href="${escapeHtml(p.downloadUrl)}" class="btn btn-secondary btn-sm" download>
            Download
          </a>
        </td>
      </tr>
    `;
  }).join("");
}

function filterPackages(term) {
  const q = term.toLowerCase().trim();
  if (!q) {
    renderPackages(allPackages);
    return;
  }
  const filtered = allPackages.filter(p =>
    p.packageName.toLowerCase().includes(q) ||
    (p.kernelVersion && p.kernelVersion.toLowerCase().includes(q)) ||
    p.version.toLowerCase().includes(q)
  );
  renderPackages(filtered);
}

async function fetchBuilds() {
  try {
    const res = await fetch("/api/builds");
    const builds = await res.json();
    renderBuilds(builds);
  } catch (err) {
    console.error("Failed to fetch builds:", err);
  }
}

function renderBuilds(builds) {
  const tbody = document.getElementById("builds-table");
  if (!builds || builds.length === 0) {
    tbody.innerHTML = `<tr><td colspan="5" style="text-align: center; color: var(--text-muted);">No recent build jobs</td></tr>`;
    return;
  }

  tbody.innerHTML = builds.map(b => {
    const statusClass = `status-${b.status}`;
    const startTime = b.startTime ? new Date(b.startTime).toLocaleTimeString() : "-";
    const duration = b.durationSec > 0 ? `${b.durationSec}s` : "-";

    return `
      <tr>
        <td><code>${escapeHtml(b.name)}</code></td>
        <td><span class="status-pill ${statusClass}">${escapeHtml(b.status)}</span></td>
        <td style="color: var(--text-secondary);">${startTime}</td>
        <td>${duration}</td>
        <td>
          <button class="btn btn-secondary btn-sm" onclick="openLogs('${escapeHtml(b.name)}')">
            Live Logs
          </button>
        </td>
      </tr>
    `;
  }).join("");
}

async function triggerBuild() {
  const btn = document.getElementById("btn-submit-build");
  btn.disabled = true;
  btn.textContent = "Starting job...";

  const kernel = document.getElementById("kernel-input").value.trim();
  const variant = document.getElementById("variant-select").value;
  const force = document.getElementById("force-build").checked;

  try {
    const res = await fetch("/api/builds", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        kernelVersion: kernel,
        variant: variant,
        forceBuild: force,
      }),
    });

    if (!res.ok) {
      const errText = await res.text();
      alert(`Failed to trigger build: ${errText}`);
    } else {
      const newJob = await res.json();
      await fetchBuilds();
      openLogs(newJob.name);
    }
  } catch (err) {
    alert(`Error: ${err.message}`);
  } finally {
    btn.disabled = false;
    btn.innerHTML = `
      <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <polygon points="5 3 19 12 5 21 5 3"></polygon>
      </svg>
      Start Build Job
    `;
  }
}

function openLogs(jobName) {
  const modal = document.getElementById("log-modal");
  const title = document.getElementById("modal-job-title");
  const term = document.getElementById("terminal-content");

  title.textContent = `Logs: ${jobName}`;
  term.textContent = "Connecting to log stream...\n";
  modal.classList.add("active");

  if (eventSource) {
    eventSource.close();
  }

  eventSource = new EventSource(`/api/builds/${encodeURIComponent(jobName)}/logs`);

  eventSource.onmessage = (event) => {
    term.textContent += event.data + "\n";
    term.scrollTop = term.scrollHeight;
  };

  eventSource.onerror = () => {
    term.textContent += "\n[Stream disconnected or job completed]\n";
    eventSource.close();
    eventSource = null;
  };
}

function closeLogModal() {
  document.getElementById("log-modal").classList.remove("active");
  if (eventSource) {
    eventSource.close();
    eventSource = null;
  }
  fetchPackages();
  fetchBuilds();
}

function copySetup() {
  const snippet = `[zfslocal]\nSigLevel = Optional TrustAll\nServer = https://pkg.markusressel.de/$repo/$arch`;
  navigator.clipboard.writeText(snippet).then(() => {
    alert("Copied pacman.conf configuration to clipboard!");
  });
}

function escapeHtml(str) {
  if (!str) return "";
  return str.replace(/[&<>'"]/g, tag => ({
    '&': '&amp;',
    '<': '&lt;',
    '>': '&gt;',
    "'": '&#39;',
    '"': '&quot;'
  }[tag] || tag));
}
