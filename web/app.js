const state = {
  recommendations: [],
  workloads: [],
  namespace: "all",
  selectedKey: "",
  statuses: new Set(["over_provisioned", "under_provisioned", "low_confidence", "balanced"]),
};

const els = {
  apiBase: document.querySelector("#apiBase"),
  refreshButton: document.querySelector("#refreshButton"),
  namespaceList: document.querySelector("#namespaceList"),
  searchInput: document.querySelector("#searchInput"),
  body: document.querySelector("#recommendationsBody"),
  workloadsBody: document.querySelector("#workloadsBody"),
  detailPanel: document.querySelector("#detailPanel"),
  podsAnalyzed: document.querySelector("#podsAnalyzed"),
  cpuSavings: document.querySelector("#cpuSavings"),
  memorySavings: document.querySelector("#memorySavings"),
  avgConfidence: document.querySelector("#avgConfidence"),
  lastUpdated: document.querySelector("#lastUpdated"),
};

const statusLabels = {
  over_provisioned: "Over provisioned",
  under_provisioned: "Under provisioned",
  low_confidence: "Low confidence",
  balanced: "Balanced",
};

const reasonLabels = {
  over_provisioned: "Bu pod için ayrılan kaynaklar, gözlenen kullanım ve tahmin edilen ihtiyacın üzerinde. Önerilen değerlere düşürmek güvenli bir kaynak optimizasyonu sağlar; amaç podu kısmak değil, boşa ayrılmış kapasiteyi geri kazanmaktır.",
  under_provisioned: "Tahmin edilen ihtiyaç mevcut request değerinden yüksek. Bu durumda kaynak azaltmak yerine request/limit değerlerini artırmak gerekir; aksi halde yoğunlukta throttling veya bellek baskısı oluşabilir.",
  low_confidence: "Model güven skoru düşük. Sistem öneriyi gösterebilir, fakat karar vermeden önce daha uzun süre metrik toplanması veya workload davranışının tekrar gözlenmesi gerekir.",
  balanced: "Mevcut kaynak istekleri önerilen değerlere yakın. Bu pod için agresif bir değişiklik önerilmiyor.",
};

const severityLabels = {
  critical: "Critical",
  high: "High",
  medium: "Medium",
  low: "Low",
};

function formatCPU(value) {
  if (!Number.isFinite(value)) return "0m";
  return `${Math.round(value * 1000)}m`;
}

function formatCores(value) {
  if (!Number.isFinite(value)) return "0";
  return `${value.toFixed(3)} core`;
}

function formatMemory(value) {
  if (!Number.isFinite(value)) return "0 Mi";
  if (Math.abs(value) >= 1024) return `${(value / 1024).toFixed(2)} Gi`;
  return `${Math.round(value)} Mi`;
}

function formatPercent(value) {
  if (!Number.isFinite(value)) return "0%";
  return `${Math.round(value * 100)}%`;
}

function labelStatus(status) {
  return statusLabels[status] || "Bilinmiyor";
}

function labelReason(rec) {
  return reasonLabels[rec.status] || rec.reason || "";
}

function labelSeverity(severity) {
  return severityLabels[severity] || "Low";
}

function recommendationKey(rec) {
  return `${rec.namespace}/${rec.pod_name}`;
}

function getFilteredRecommendations() {
  const query = els.searchInput.value.trim().toLowerCase();
  return state.recommendations.filter((rec) => {
    const key = recommendationKey(rec).toLowerCase();
    const namespaceMatch = state.namespace === "all" || rec.namespace === state.namespace;
    const statusMatch = state.statuses.has(rec.status);
    const queryMatch = !query || key.includes(query);
    return namespaceMatch && statusMatch && queryMatch;
  });
}

function getFilteredWorkloads() {
  const query = els.searchInput.value.trim().toLowerCase();
  return state.workloads.filter((workload) => {
    const key = `${workload.namespace}/${workload.workload_type}/${workload.workload_name}`.toLowerCase();
    const namespaceMatch = state.namespace === "all" || workload.namespace === state.namespace;
    const statusMatch = state.statuses.has(workload.status);
    const queryMatch = !query || key.includes(query);
    return namespaceMatch && statusMatch && queryMatch;
  });
}

function renderNamespaces() {
  const namespaces = ["all", ...new Set(state.recommendations.map((rec) => rec.namespace).sort())];
  els.namespaceList.innerHTML = namespaces
    .map((namespace) => {
      const label = namespace === "all" ? "All namespaces" : namespace;
      const displayLabel = namespace === "all" ? "All namespaces" : namespace;
      const active = namespace === state.namespace ? "active" : "";
      return `<button class="namespace-button ${active}" type="button" data-namespace="${namespace}">${displayLabel}</button>`;
    })
    .join("");
}

function renderSummary() {
  const recs = getFilteredRecommendations();
  const cpuSavings = recs.reduce((sum, rec) => sum + (rec.impact?.cpu_savings_cores || 0), 0);
  const cpuAdditional = recs.reduce((sum, rec) => sum + (rec.impact?.cpu_additional_cores || 0), 0);
  const memorySavings = recs.reduce((sum, rec) => sum + (rec.impact?.memory_savings_mb || 0), 0);
  const memoryAdditional = recs.reduce((sum, rec) => sum + (rec.impact?.memory_additional_mb || 0), 0);
  const confidence = recs.length
    ? recs.reduce((sum, rec) => sum + (rec.overall_confidence || 0), 0) / recs.length
    : 0;

  els.podsAnalyzed.textContent = recs.length;
  els.cpuSavings.textContent = cpuAdditional > cpuSavings
    ? `+${formatCores(cpuAdditional - cpuSavings)}`
    : formatCores(cpuSavings - cpuAdditional);
  els.memorySavings.textContent = memoryAdditional > memorySavings
    ? `+${formatMemory(memoryAdditional - memorySavings)}`
    : formatMemory(memorySavings - memoryAdditional);
  els.avgConfidence.textContent = formatPercent(confidence);
}

function renderTable() {
  const recs = getFilteredRecommendations();
  if (!recs.length) {
    els.body.innerHTML = `<tr><td colspan="8" class="muted">No recommendations match the current filters.</td></tr>`;
    return;
  }

  els.body.innerHTML = recs
    .map((rec) => {
      const key = recommendationKey(rec);
      const selected = key === state.selectedKey ? "selected" : "";
      const impact = rec.impact || {};
      return `
        <tr class="${selected}" data-key="${key}">
          <td>
            <div class="pod-name">
              <strong>${rec.pod_name}</strong>
              <span class="muted">${rec.namespace}</span>
            </div>
          </td>
          <td>${rec.workload_type || "Pod"} / ${rec.workload_name || rec.pod_name}</td>
          <td>${rec.node_name || "unknown"}</td>
          <td><span class="badge ${rec.status}">${labelStatus(rec.status)}</span></td>
          <td><span class="severity ${rec.severity || "low"}">${labelSeverity(rec.severity)}</span></td>
          <td>${impact.requires_additional_cpu ? "+" : ""}${impact.requires_additional_cpu ? formatCores(impact.cpu_additional_cores) : formatCores(impact.cpu_savings_cores || 0)}</td>
          <td>${impact.requires_additional_memory ? "+" : ""}${impact.requires_additional_memory ? formatMemory(impact.memory_additional_mb) : formatMemory(impact.memory_savings_mb || 0)}</td>
          <td>${formatPercent(rec.overall_confidence || 0)}</td>
        </tr>
      `;
    })
    .join("");
}

function renderWorkloads() {
  const workloads = getFilteredWorkloads();
  if (!workloads.length) {
    els.workloadsBody.innerHTML = `<tr><td colspan="7" class="muted">No workload summaries match the current filters.</td></tr>`;
    return;
  }

  els.workloadsBody.innerHTML = workloads
    .map((workload) => {
      const cpuImpact = workload.cpu_additional_cores > workload.cpu_savings_cores
        ? `+${formatCores(workload.cpu_additional_cores - workload.cpu_savings_cores)}`
        : formatCores(workload.cpu_savings_cores - workload.cpu_additional_cores);
      const memoryImpact = workload.memory_additional_mb > workload.memory_savings_mb
        ? `+${formatMemory(workload.memory_additional_mb - workload.memory_savings_mb)}`
        : formatMemory(workload.memory_savings_mb - workload.memory_additional_mb);
      return `
        <tr>
          <td>
            <div class="pod-name">
              <strong>${workload.workload_name}</strong>
              <span class="muted">${workload.namespace} / ${workload.workload_type}</span>
            </div>
          </td>
          <td><span class="badge ${workload.status}">${labelStatus(workload.status)}</span></td>
          <td><span class="severity ${workload.severity || "low"}">${labelSeverity(workload.severity)}</span></td>
          <td>${workload.pod_count}</td>
          <td>${cpuImpact}</td>
          <td>${memoryImpact}</td>
          <td>${formatPercent(workload.avg_confidence || 0)}</td>
        </tr>
      `;
    })
    .join("");
}

function renderDetail() {
  const rec = state.recommendations.find((item) => recommendationKey(item) === state.selectedKey);
  if (!rec) {
    els.detailPanel.innerHTML = `
      <div class="empty-state">
        <h3>Select a pod</h3>
        <p>Choose a recommendation to inspect container-level requests, limits, and YAML output.</p>
      </div>
    `;
    return;
  }

  const impact = rec.impact || {};
  const containers = Object.values(rec.containers || {});
  els.detailPanel.innerHTML = `
    <div class="detail-header">
      <div class="detail-title">
        <h3>${rec.pod_name}</h3>
        <span class="badge ${rec.status}">${labelStatus(rec.status)}</span>
      </div>
      <span class="muted">${rec.namespace}</span>
      <span class="muted">${rec.workload_type || "Pod"}: ${rec.workload_name || rec.pod_name} / Node: ${rec.node_name || "unknown"}</span>
      <span class="severity ${rec.severity || "low"}">Intervention priority: ${labelSeverity(rec.severity)}</span>
      <p>${labelReason(rec)}</p>
      <p class="recommendation-note">${buildRecommendationNote(rec)}</p>
    </div>

    <div class="detail-section">
      <div class="kv-grid">
        <div class="kv"><span>CPU savings</span><strong>${formatCores(impact.cpu_savings_cores || 0)}</strong></div>
        <div class="kv"><span>CPU additional</span><strong>${formatCores(impact.cpu_additional_cores || 0)}</strong></div>
        <div class="kv"><span>Memory savings</span><strong>${formatMemory(impact.memory_savings_mb || 0)}</strong></div>
        <div class="kv"><span>Memory additional</span><strong>${formatMemory(impact.memory_additional_mb || 0)}</strong></div>
      </div>
    </div>

    <div class="detail-section">
      ${containers.map(renderContainer).join("")}
    </div>

    <div class="detail-section">
      <span class="muted">YAML patch preview</span>
      <pre>${renderYamlPatch(rec)}</pre>
    </div>
  `;
}

function renderContainer(container) {
  return `
    <article class="container-row">
      <h4>Container: ${container.container}</h4>
      <div class="resource-grid">
        <div class="kv"><span>CPU request</span><strong>${formatCPU(container.current.cpu_request)} -> ${formatCPU(container.recommended_cpu_request)}</strong></div>
        <div class="kv"><span>CPU limit</span><strong>${formatCPU(container.current.cpu_limit)} -> ${formatCPU(container.recommended_cpu_limit)}</strong></div>
        <div class="kv"><span>Memory request</span><strong>${formatMemory(container.current.memory_request)} -> ${formatMemory(container.recommended_memory_request)}</strong></div>
        <div class="kv"><span>Memory limit</span><strong>${formatMemory(container.current.memory_limit)} -> ${formatMemory(container.recommended_memory_limit)}</strong></div>
      </div>
    </article>
  `;
}

function buildRecommendationNote(rec) {
  const impact = rec.impact || {};
  if (rec.status === "over_provisioned") {
    return `Aksiyon: CPU request yaklaşık ${formatCores(impact.cpu_savings_cores || 0)}, bellek request yaklaşık ${formatMemory(impact.memory_savings_mb || 0)} azaltılabilir. Bu pod mevcut metriklere göre kapasite fazlası taşıyor; öneri dry-run olduğu için doğrudan uygulama yapmaz.`;
  }
  if (rec.status === "under_provisioned") {
    return `Aksiyon: CPU için ${formatCores(impact.cpu_additional_cores || 0)}, bellek için ${formatMemory(impact.memory_additional_mb || 0)} ek kaynak ayrılması önerilir. Bu durum HPA ölçekleme kararı değil, pod başına request/limit düzeltmesidir.`;
  }
  if (rec.status === "low_confidence") {
    return "Aksiyon: Öneri düşük güvenle üretildiği için hemen uygulanmamalı. Daha fazla Prometheus verisi toplandıktan sonra tekrar analiz edilmelidir.";
  }
  return "Aksiyon: Pod dengeli görünüyor; büyük bir kaynak değişikliği önerilmiyor.";
}

function renderYamlPatch(rec) {
  return Object.values(rec.containers || {})
    .map((container) => [
      `# ${container.container}: ${buildYamlComment(container)}`,
      `- name: ${container.container}`,
      "  resources:",
      "    requests:",
      `      cpu: "${Math.round(container.recommended_cpu_request * 1000)}m"`,
      `      memory: "${container.recommended_memory_request}Mi"`,
      "    limits:",
      `      cpu: "${Math.round(container.recommended_cpu_limit * 1000)}m"`,
      `      memory: "${container.recommended_memory_limit}Mi"`,
    ].join("\n"))
    .join("\n\n");
}

function buildYamlComment(container) {
  const cpuRequestDelta = container.current.cpu_request - container.recommended_cpu_request;
  const memoryRequestDelta = container.current.memory_request - container.recommended_memory_request;

  if (cpuRequestDelta >= 0 && memoryRequestDelta >= 0) {
    return `mevcut metriklere göre CPU request ${formatCPU(cpuRequestDelta)}, bellek request ${formatMemory(memoryRequestDelta)} azaltılabilir.`;
  }
  const additions = [];
  if (cpuRequestDelta < 0) additions.push(`CPU request ${formatCPU(Math.abs(cpuRequestDelta))} artırılmalı`);
  if (memoryRequestDelta < 0) additions.push(`bellek request ${formatMemory(Math.abs(memoryRequestDelta))} artırılmalı`);
  return `${additions.join(", ")}.`;
}

function render() {
  renderNamespaces();
  renderSummary();
  renderWorkloads();
  renderTable();
  renderDetail();
}

async function loadRecommendations() {
  const base = els.apiBase.value.replace(/\/$/, "");
  els.lastUpdated.textContent = "Loading...";

  try {
    const response = await fetch(`${base}/api/v1/recommendations`);
    if (!response.ok) throw new Error(`API returned ${response.status}`);
    const data = await response.json();
    const workloadResponse = await fetch(`${base}/api/v1/workloads`);
    if (!workloadResponse.ok) throw new Error(`Workload API returned ${workloadResponse.status}`);
    const workloadData = await workloadResponse.json();
    state.recommendations = Array.isArray(data.recommendations) ? data.recommendations : [];
    state.workloads = Array.isArray(workloadData.workloads) ? workloadData.workloads : [];
    if (!state.recommendations.some((rec) => recommendationKey(rec) === state.selectedKey)) {
      state.selectedKey = state.recommendations[0] ? recommendationKey(state.recommendations[0]) : "";
    }
    els.lastUpdated.textContent = `Updated ${new Date().toLocaleTimeString("tr-TR")}`;
    els.lastUpdated.classList.remove("error");
    render();
  } catch (error) {
    els.lastUpdated.textContent = `API unavailable: ${error.message}`;
    els.lastUpdated.classList.add("error");
    state.recommendations = [];
    render();
  }
}

els.refreshButton.addEventListener("click", loadRecommendations);
els.searchInput.addEventListener("input", render);

els.namespaceList.addEventListener("click", (event) => {
  const button = event.target.closest("[data-namespace]");
  if (!button) return;
  state.namespace = button.dataset.namespace;
  render();
});

document.querySelectorAll(".status-filters input").forEach((input) => {
  input.addEventListener("change", () => {
    if (input.checked) {
      state.statuses.add(input.value);
    } else {
      state.statuses.delete(input.value);
    }
    render();
  });
});

els.body.addEventListener("click", (event) => {
  const row = event.target.closest("[data-key]");
  if (!row) return;
  state.selectedKey = row.dataset.key;
  render();
});

loadRecommendations();
setInterval(loadRecommendations, 30000);
