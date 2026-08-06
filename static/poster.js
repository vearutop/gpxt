(function () {
    "use strict";

    var config = window.POSTER_CONFIG;
    // Empty for the local `poster` command (assets at "/"); a per-session prefix like
    // "/poster/abc123" in server mode. Explicit concatenation instead of relative URLs —
    // relying on trailing-slash resolution is a classic footgun.
    var base = config.basePath || "";
    var map = null;
    var statsData = null;
    var allGeoJson = [];   // indexed by file, filled as fetches complete
    var trackCanvas = null;
    var tc = null;         // 2D context of the track overlay canvas
    var palette = ["#f97316", "#22c55e", "#06b6d4", "#eab308", "#ef4444", "#8b5cf6"];
    var fillLayers = {};   // layerId → {prop, original} — cached before first strip
    var trackBounds = null; // stored for re-fit after bearing change
    var trackColors = config.files.map(function (_, i) { return palette[i % palette.length]; });
    var posterTitle = "";
    var statsOnTop = false;
    var trackProfiles = []; // one entry per file, fetched from /profiles.json

    // Single source of truth for the selectable mini-charts: add an entry here to make a
    // new metric's chart available in the toolbar and on the poster at once. "key" must
    // match a field in the /profiles.json response.
    var MINICHART_DEFS = [
        { key: "elevation", label: "Elevation",   unit: "m" },
        { key: "hr",        label: "Heart Rate",  unit: "bpm" },
        { key: "power",     label: "Power",       unit: "W" },
        { key: "atemp",     label: "Temperature", unit: "°C" },
        { key: "speed",     label: "Speed",       unit: "km/h" },
    ];

    // Per-chart runtime state — box position/size as % of frame, independent sliders
    // standing in for drag-and-resize.
    var miniCharts = {};
    MINICHART_DEFS.forEach(function (d) {
        miniCharts[d.key] = { enabled: false, caption: d.label, bg: 35, x: 0, y: 80, w: 100, h: 20 };
    });

    // Single source of truth for selectable poster stats: add an entry here to make a
    // new metric available in the toolbar, live overlay, and PNG export all at once.
    var STAT_DEFS = [
        { key: "distance",   toggleLabel: "Dist",  gridLabel: "Distance",    default: true,
          value: function (a) { return a.distanceKm.toFixed(1) + " km"; } },
        { key: "movingTime", toggleLabel: "Time",   gridLabel: "Moving Time", default: true,
          value: function (a) { return a.movingTime; } },
        { key: "avgSpeed",   toggleLabel: "Speed",  gridLabel: "Avg Speed",   default: true,
          value: function (a) { return a.avgSpeedKmh.toFixed(1) + " km/h"; } },
        { key: "maxSpeed",   toggleLabel: "Max",    gridLabel: "Max Speed",   default: false,
          value: function (a) { return a.maxSpeedKmh.toFixed(1) + " km/h"; } },
        { key: "elevation",  toggleLabel: "Elev",   gridLabel: "Elevation",   default: true,
          value: function (a) { return "↑ " + Math.round(a.uphillM) + " m"; } },
        { key: "descent",    toggleLabel: "Desc",   gridLabel: "Descent",     default: false,
          value: function (a) { return "↓ " + Math.round(a.downhillM) + " m"; } },
    ];

    var enabledStats = {};
    STAT_DEFS.forEach(function (d) { enabledStats[d.key] = d.default; });

    // Per-stat manual override (e.g. Strava's corrected elevation instead of gpxt's own
    // computed value) — non-empty string wins over the computed value verbatim.
    var overrides = {};

    function init() {
        loadBackgroundImage()
            .then(function () {
                initTrackCanvas();
                initMap();
                return fetch(base + "/stats.json").then(function (r) { return r.json(); });
            })
            .then(function (stats) {
                statsData = stats;
                renderStatsOverlay(stats);
                setupControls();

                fetch(base + "/profiles.json").then(function (r) { return r.json(); }).then(function (profiles) {
                    trackProfiles = profiles;
                    drawTracks();
                });
            })
            .catch(function (err) {
                document.getElementById("poster-frame").textContent = "Error: " + err.message;
            });
    }

    function loadBackgroundImage() {
        return new Promise(function (resolve, reject) {
            var img = document.getElementById("bg-image");
            img.onload = function () {
                var frame = document.getElementById("poster-frame");
                var ratio = img.naturalWidth / img.naturalHeight;
                var wrapper = document.getElementById("poster-wrapper");
                var wrapperH = wrapper.clientHeight - 48;
                var wrapperW = wrapper.clientWidth - 48;
                var h, w;
                if (wrapperW / ratio <= wrapperH) {
                    w = wrapperW;
                    h = Math.round(w / ratio);
                } else {
                    h = wrapperH;
                    w = Math.round(h * ratio);
                }
                frame.style.width = w + "px";
                frame.style.height = h + "px";
                frame.getBoundingClientRect();
                resolve();
            };
            img.onerror = function () { reject(new Error("Failed to load background image")); };
            img.src = base + "/image";
        });
    }

    function initTrackCanvas() {
        trackCanvas = document.getElementById("track-canvas");
        var frame = document.getElementById("poster-frame");
        var dpr = window.devicePixelRatio || 1;
        // Physical pixels for the canvas buffer; CSS size matches the frame.
        trackCanvas.width  = Math.round(frame.clientWidth  * dpr);
        trackCanvas.height = Math.round(frame.clientHeight * dpr);
        trackCanvas.style.width  = frame.clientWidth  + "px";
        trackCanvas.style.height = frame.clientHeight + "px";
        tc = trackCanvas.getContext("2d");
    }

    function initMap() {
        map = new maplibregl.Map({
            container: "map",
            style: config.styleURL,
            center: [0, 0],
            zoom: 1,
            preserveDrawingBuffer: true,
            attributionControl: false,
        });

        map.on("load", function () {
            initFillLayers(); // must come before stripMapFills
            stripMapFills();
            map.resize();
            loadTracks();
        });


        // Redraw the track overlay whenever MapLibre repaints.
        map.on("render", drawTracks);

        var opacity = document.getElementById("opacity-slider").value / 100;
        map.getContainer().style.opacity = String(opacity);
    }

    var FILL_OPACITY_PROP = {
        "background":    "background-opacity",
        "fill":          "fill-opacity",
        "fill-extrusion":"fill-extrusion-opacity",
        "raster":        "raster-opacity",
    };

    // Cache original values from the style spec before we touch anything.
    function initFillLayers() {
        map.getStyle().layers.forEach(function (layer) {
            var prop = FILL_OPACITY_PROP[layer.type];
            if (!prop) return;
            var paint = layer.paint || {};
            fillLayers[layer.id] = {
                prop:     prop,
                original: paint[prop] !== undefined ? paint[prop] : 1,
            };
        });
    }

    function stripMapFills() {
        Object.keys(fillLayers).forEach(function (id) {
            map.setPaintProperty(id, fillLayers[id].prop, 0);
        });
    }

    function restoreMapFills() {
        Object.keys(fillLayers).forEach(function (id) {
            map.setPaintProperty(id, fillLayers[id].prop, fillLayers[id].original);
        });
    }

    function loadTracks() {
        var bounds = new maplibregl.LngLatBounds();
        var loaded = 0;

        config.files.forEach(function (name, i) {
            fetch(base + "/track/" + i + ".geojson")
                .then(function (r) { return r.json(); })
                .then(function (geojson) {
                    allGeoJson[i] = geojson;

                    // Update the color picker label with the real track name.
                    var trackName = "";
                    geojson.features.some(function (f) {
                        if (f.properties && f.properties.name) { trackName = f.properties.name; return true; }
                    });
                    var labelEl = document.getElementById("track-label-" + i);
                    if (labelEl && trackName) labelEl.textContent = trackName;

                    geojson.features.forEach(function (f) {
                        if (f.geometry.type === "LineString") {
                            f.geometry.coordinates.forEach(function (c) { bounds.extend(c); });
                        } else if (f.geometry.type === "Point") {
                            bounds.extend(f.geometry.coordinates);
                        }
                    });

                    loaded++;
                    if (loaded === config.files.length) {
                        if (!bounds.isEmpty()) {
                            trackBounds = bounds;
                            map.fitBounds(bounds, { padding: 56, duration: 0 });
                        }
                        map.once("idle", function () {
                            var mc = map.getCanvas();
                            document.getElementById("export-size").textContent =
                                mc.width + "×" + mc.height + " px";
                            document.getElementById("save-btn").disabled = false;
                            document.getElementById("auto-rotate-btn").disabled = false;
                        });
                    }
                });
        });
    }

    // Redraws all tracks on the overlay canvas using projected screen coordinates.
    // Called on every MapLibre render so the overlay stays in sync with pan/zoom.
    function drawTracks() {
        if (!tc) return;
        var dpr = window.devicePixelRatio || 1;
        tc.clearRect(0, 0, trackCanvas.width, trackCanvas.height);

        allGeoJson.forEach(function (geojson, i) {
            if (!geojson) return;
            var color = trackColors[i] || palette[i % palette.length];

            geojson.features.forEach(function (feature) {
                if (!feature.geometry) return;

                if (feature.geometry.type === "LineString") {
                    var coords = feature.geometry.coordinates;
                    if (coords.length < 2) return;
                    drawLine(coords, "rgba(0,0,0,0.45)", 8 * dpr);
                    drawLine(coords, color, 5 * dpr);

                } else if (feature.geometry.type === "Point") {
                    var pt = map.project(feature.geometry.coordinates);
                    var r = 6 * dpr;
                    tc.beginPath();
                    tc.arc(pt.x * dpr, pt.y * dpr, r, 0, Math.PI * 2);
                    tc.fillStyle = color;
                    tc.fill();
                    tc.strokeStyle = "#ffffff";
                    tc.lineWidth = 2 * dpr;
                    tc.stroke();
                }
            });
        });

        drawMiniCharts();
    }

    function drawMiniCharts() {
        MINICHART_DEFS.forEach(drawMiniChart);
    }

    // Draws one metric's profile chart onto the same overlay canvas as the tracks, so it
    // rides along for free in the PNG export (which copies that canvas wholesale).
    function drawMiniChart(def) {
        var state = miniCharts[def.key];
        if (!state.enabled || !trackProfiles.length) return;

        var seriesPerFile = trackProfiles.map(function (tp) { return tp[def.key] || []; });
        if (!seriesPerFile.some(function (s) { return s.length; })) return;

        var dpr = window.devicePixelRatio || 1;
        var cw = trackCanvas.width, ch = trackCanvas.height;

        var w = (state.w / 100) * cw;
        var h = (state.h / 100) * ch;
        if (w <= 0 || h <= 0) return;

        // X/Y slide the box between flush-against-the-near-edge (0%) and
        // flush-against-the-far-edge (100%) over the remaining slack space, so the box
        // never runs off-canvas and 100% always means "flush right/bottom", not "clamped
        // into nothing".
        var rect = {
            x: (state.x / 100) * (cw - w),
            y: (state.y / 100) * (ch - h),
            w: w,
            h: h,
        };

        var minV = Infinity, maxV = -Infinity, totalDist = 0;
        var offsets = seriesPerFile.map(function (points) {
            var offset = totalDist;
            points.forEach(function (pt) {
                if (pt.v < minV) minV = pt.v;
                if (pt.v > maxV) maxV = pt.v;
            });
            var last = points[points.length - 1];
            totalDist += last ? last.d : 0;
            return offset;
        });
        if (!isFinite(minV) || totalDist === 0) return;
        if (maxV === minV) maxV = minV + 1;

        var padY = rect.h * 0.12;

        function xAt(globalD) { return rect.x + (globalD / totalDist) * rect.w; }
        function yAt(v) { return rect.y + rect.h - padY - ((v - minV) / (maxV - minV)) * (rect.h - 2 * padY); }

        tc.fillStyle = "rgba(0,0,0," + (state.bg / 100) + ")";
        tc.fillRect(rect.x, rect.y, rect.w, rect.h);

        // Ruler: min/mid/max gridlines, drawn under the profile lines. Labeled on both
        // edges so the value is readable regardless of which side the profile crowds.
        var padX = 6 * dpr;
        tc.font = Math.round(10 * dpr) + "px system-ui,-apple-system,sans-serif";
        tc.textBaseline = "middle";
        [minV, (minV + maxV) / 2, maxV].forEach(function (v) {
            var y = yAt(v);
            var label = Math.round(v) + def.unit;

            tc.strokeStyle = "rgba(255,255,255,0.15)";
            tc.lineWidth = Math.max(1, Math.round(dpr));
            tc.beginPath();
            tc.moveTo(rect.x, y);
            tc.lineTo(rect.x + rect.w, y);
            tc.stroke();

            tc.fillStyle = "rgba(255,255,255,0.55)";
            tc.textAlign = "left";
            tc.fillText(label, rect.x + padX, y);
            tc.textAlign = "right";
            tc.fillText(label, rect.x + rect.w - padX, y);
        });

        seriesPerFile.forEach(function (points, i) {
            if (!points.length) return;

            tc.beginPath();
            points.forEach(function (pt, j) {
                var x = xAt(offsets[i] + pt.d);
                var y = yAt(pt.v);
                if (j === 0) tc.moveTo(x, y); else tc.lineTo(x, y);
            });
            tc.strokeStyle = trackColors[i] || palette[i % palette.length];
            tc.lineWidth = 2 * dpr;
            tc.lineJoin = "round";
            tc.stroke();

            if (i > 0) {
                var dividerX = xAt(offsets[i]);
                tc.strokeStyle = "rgba(255,255,255,0.25)";
                tc.lineWidth = 1 * dpr;
                tc.beginPath();
                tc.moveTo(dividerX, rect.y);
                tc.lineTo(dividerX, rect.y + rect.h);
                tc.stroke();
            }
        });

        if (state.caption) {
            tc.font = "bold " + Math.round(11 * dpr) + "px system-ui,-apple-system,sans-serif";
            tc.textAlign = "right";
            tc.textBaseline = "top";
            tc.fillStyle = "rgba(255,255,255,0.85)";
            tc.fillText(state.caption, rect.x + rect.w - padX, rect.y + 6 * dpr);
        }

        tc.textAlign = "left";
        tc.textBaseline = "alphabetic";
    }

    function drawLine(coords, color, width) {
        var dpr = window.devicePixelRatio || 1;
        tc.beginPath();
        tc.strokeStyle = color;
        tc.lineWidth = width;
        tc.lineCap = "round";
        tc.lineJoin = "round";
        coords.forEach(function (coord, j) {
            var pt = map.project(coord);
            if (j === 0) tc.moveTo(pt.x * dpr, pt.y * dpr);
            else         tc.lineTo(pt.x * dpr, pt.y * dpr);
        });
        tc.stroke();
    }

    function renderStatsOverlay(stats) {
        if (!stats || stats.length === 0) return;

        var agg = aggregateStats(stats);
        var overlay = document.getElementById("stats-overlay");
        var displayName = posterTitle || agg.name;

        var itemsHtml = STAT_DEFS
            .filter(function (d) { return enabledStats[d.key]; })
            .map(function (d) { return statItem(overrides[d.key] || d.value(agg), d.gridLabel); });

        overlay.innerHTML =
            '<div class="stats-name">' + esc(displayName) + "</div>" +
            '<div class="stats-date">' + esc(agg.date) + "</div>" +
            (itemsHtml.length
                ? '<div class="stats-grid" style="grid-template-columns:repeat(' + itemsHtml.length + ',1fr)">' +
                  itemsHtml.join("") + "</div>"
                : "");
    }

    function aggregateStats(stats) {
        var totalDist = 0, totalSecs = 0, totalUphill = 0, totalDownhill = 0, maxSpeed = 0;
        stats.forEach(function (s) {
            totalDist     += s.distance_km;
            totalSecs     += s.moving_time_secs;
            totalUphill   += s.uphill_m;
            totalDownhill += s.downhill_m;
            maxSpeed = Math.max(maxSpeed, s.max_speed_kmh);
        });
        return {
            name:        stats[0].name,
            date:        dateRange(stats),
            distanceKm:  totalDist,
            movingTime:  fmtDuration(totalSecs),
            avgSpeedKmh: totalSecs > 0 ? (totalDist / totalSecs) * 3600 : 0,
            maxSpeedKmh: maxSpeed,
            uphillM:     totalUphill,
            downhillM:   totalDownhill,
        };
    }

    // Renders a single date for one day, or "Month D–D, YYYY" / "Month D, YYYY – Month D, YYYY"
    // spanning multiple files' start dates.
    function dateRange(stats) {
        var dates = stats
            .map(function (s) { return s.date_start ? new Date(s.date_start) : null; })
            .filter(function (d) { return d && !isNaN(d); })
            .sort(function (a, b) { return a - b; });

        if (dates.length === 0) return stats[0].date || "";

        var first = dates[0], last = dates[dates.length - 1];
        var fmtFull  = function (d) { return d.toLocaleDateString("en-US", { month: "long", day: "numeric", year: "numeric" }); };
        var fmtMonth = function (d) { return d.toLocaleDateString("en-US", { month: "long" }); };

        if (first.toDateString() === last.toDateString()) return fmtFull(first);
        if (first.getFullYear() === last.getFullYear() && first.getMonth() === last.getMonth()) {
            return fmtMonth(first) + " " + first.getDate() + "–" + last.getDate() + ", " + first.getFullYear();
        }
        return fmtFull(first) + " – " + fmtFull(last);
    }

    function fmtDuration(secs) {
        var h = Math.floor(secs / 3600);
        var m = Math.floor((secs % 3600) / 60);
        if (h > 0) return h + "h " + String(m).padStart(2, "0") + "m";
        return m + "m";
    }

    function statItem(value, label) {
        return (
            '<div class="stat-item">' +
            '<div class="stat-value">' + value + "</div>" +
            '<div class="stat-label">' + label + "</div>" +
            "</div>"
        );
    }

    // Fixed-width so the toolbar doesn't jitter as the digit count changes across the
    // slider's -180..180 range (e.g. "5°" vs "-180°").
    function fmtBearing(b) {
        var sign = b < 0 ? "-" : " ";
        return sign + Math.abs(b).toString().padStart(3, "0") + "°";
    }

    function esc(s) {
        return String(s || "")
            .replace(/&/g, "&amp;")
            .replace(/</g, "&lt;")
            .replace(/>/g, "&gt;");
    }

    // PCA-based auto-rotate: finds the principal axis of the track points and
    // aligns it with the poster's longer dimension (vertical for portrait, etc.).
    function autoRotate() {
        var coords = [];
        allGeoJson.forEach(function (geojson) {
            if (!geojson) return;
            geojson.features.forEach(function (f) {
                if (f.geometry && f.geometry.type === "LineString") {
                    f.geometry.coordinates.forEach(function (c) {
                        coords.push([c[0], c[1]]);
                    });
                }
            });
        });
        if (coords.length < 2) return;

        // Centroid.
        var mx = 0, my = 0;
        coords.forEach(function (c) { mx += c[0]; my += c[1]; });
        mx /= coords.length;
        my /= coords.length;

        // Scale longitude by cos(lat) so distances are isotropic.
        var cosLat = Math.cos(my * Math.PI / 180);

        // Covariance matrix elements.
        var cxx = 0, cyy = 0, cxy = 0;
        coords.forEach(function (c) {
            var dx = (c[0] - mx) * cosLat;
            var dy =  c[1] - my;
            cxx += dx * dx;
            cyy += dy * dy;
            cxy += dx * dy;
        });

        // Angle of the principal eigenvector (atan2 trick for 2×2 symmetric matrix).
        var angle = Math.atan2(2 * cxy, cxx - cyy) / 2; // radians, from East axis

        // Convert to MapLibre bearing (degrees clockwise from North).
        var bearing = Math.atan2(Math.cos(angle), Math.sin(angle)) * 180 / Math.PI;

        // For a landscape frame the principal axis should be horizontal → rotate 90°.
        var frame = document.getElementById("poster-frame");
        if (frame.clientWidth > frame.clientHeight) {
            bearing = (bearing + 90 + 360) % 360;
        }

        // Normalise to (−180, 180].
        if (bearing > 180) bearing -= 360;

        if (trackBounds) {
            map.fitBounds(trackBounds, { padding: 56, duration: 300, bearing: bearing });
        } else {
            map.setBearing(bearing);
        }
        document.getElementById("bearing-slider").value = Math.round(bearing);
        document.getElementById("bearing-value").textContent = fmtBearing(Math.round(bearing));
    }

    function initTrackColorPickers() {
        var container = document.getElementById("track-colors");
        config.files.forEach(function (fileName, i) {
            var label = document.createElement("label");
            label.className = "toolbar-control";

            var swatch = document.createElement("input");
            swatch.type = "color";
            swatch.value = trackColors[i];
            swatch.className = "track-color-input";
            swatch.addEventListener("input", function () {
                trackColors[i] = this.value;
                drawTracks();
            });

            var nameSpan = document.createElement("span");
            nameSpan.id = "track-label-" + i;
            nameSpan.className = "toolbar-control__label";
            // Filename as placeholder until GeoJSON loads.
            nameSpan.textContent = fileName.split(/[/\\]/).pop().replace(/\.[^.]+$/, "");

            label.appendChild(swatch);
            label.appendChild(nameSpan);
            container.appendChild(label);
        });
    }

    // Builds one <details> popover per mini-chart metric, each with the same
    // show/caption/background/position/size controls. All wired generically via
    // data-mc (chart key) / data-field (state field) attributes instead of repeating
    // markup+listeners per metric.
    function initMiniChartControls() {
        var container = document.getElementById("minichart-controls");

        container.innerHTML = MINICHART_DEFS.map(function (def) {
            var k = esc(def.key);
            var s = miniCharts[def.key];

            function row(label, field, inputHtml, unit) {
                return (
                    '<label class="toolbar-control">' +
                    '<span class="toolbar-control__label">' + esc(label) + "</span>" +
                    inputHtml +
                    (unit ? '<span data-mc-value="' + k + ":" + field + '">' + s[field] + unit + "</span>" : "") +
                    "</label>"
                );
            }

            return (
                '<details class="toolbar-popover">' +
                '<summary class="toolbar-popover__trigger">' + esc(def.label) + " chart ▾</summary>" +
                '<div class="toolbar-popover__panel">' +
                '<label class="toolbar-control">' +
                '<input type="checkbox" data-mc="' + k + '" data-field="enabled">' +
                '<span class="toolbar-control__label">Show chart</span>' +
                "</label>" +
                row("Caption", "caption",
                    '<input type="text" class="toolbar-text-input" data-mc="' + k + '" data-field="caption" value="' + esc(s.caption) + '">') +
                row("Background", "bg",
                    '<input type="range" min="0" max="100" value="' + s.bg + '" data-mc="' + k + '" data-field="bg">', "%") +
                row("Horizontal", "x",
                    '<input type="range" min="0" max="100" value="' + s.x + '" data-mc="' + k + '" data-field="x">', "%") +
                row("Vertical", "y",
                    '<input type="range" min="0" max="100" value="' + s.y + '" data-mc="' + k + '" data-field="y">', "%") +
                row("Width", "w",
                    '<input type="range" min="5" max="100" value="' + s.w + '" data-mc="' + k + '" data-field="w">', "%") +
                row("Height", "h",
                    '<input type="range" min="5" max="100" value="' + s.h + '" data-mc="' + k + '" data-field="h">', "%") +
                "</div></details>"
            );
        }).join("");

        container.querySelectorAll("[data-mc]").forEach(function (el) {
            var key = el.dataset.mc, field = el.dataset.field;
            var eventName = el.type === "checkbox" ? "change" : "input";

            el.addEventListener(eventName, function () {
                var val = el.type === "checkbox" ? el.checked
                    : el.type === "range" ? Number(el.value)
                    : el.value;

                miniCharts[key][field] = val;

                if (el.type === "range") {
                    var valueEl = container.querySelector('[data-mc-value="' + key + ":" + field + '"]');
                    if (valueEl) valueEl.textContent = val + "%";
                }

                drawTracks();
            });
        });
    }

    function initStatToggles() {
        var container = document.getElementById("stat-toggles");
        STAT_DEFS.forEach(function (d) {
            var item = document.createElement("div");
            item.className = "toolbar-stat-item";

            var label = document.createElement("label");
            label.className = "toolbar-stat";

            var checkbox = document.createElement("input");
            checkbox.type = "checkbox";
            checkbox.checked = d.default;
            checkbox.addEventListener("change", function () {
                enabledStats[d.key] = checkbox.checked;
                if (statsData) renderStatsOverlay(statsData);
            });

            label.appendChild(checkbox);
            label.appendChild(document.createTextNode(d.toggleLabel));

            // Outside the <label> so clicking/typing here doesn't toggle the checkbox.
            var override = document.createElement("input");
            override.type = "text";
            override.className = "toolbar-stat-override";
            override.placeholder = "custom";
            override.title = "Override the computed " + d.gridLabel + " value";
            override.addEventListener("input", function () {
                overrides[d.key] = override.value;
                if (statsData) renderStatsOverlay(statsData);
            });

            item.appendChild(label);
            item.appendChild(override);
            container.appendChild(item);
        });
    }

    function setupControls() {
        initTrackColorPickers();
        initStatToggles();
        initMiniChartControls();
        var slider     = document.getElementById("opacity-slider");
        var opacityVal = document.getElementById("opacity-value");

        slider.addEventListener("input", function () {
            opacityVal.textContent = slider.value + "%";
            if (map) map.getContainer().style.opacity = String(slider.value / 100);
        });

        document.getElementById("fills-checkbox").addEventListener("change", function () {
            if (this.checked) restoreMapFills();
            else              stripMapFills();
        });

        var bearingSlider = document.getElementById("bearing-slider");
        var bearingVal    = document.getElementById("bearing-value");
        bearingSlider.addEventListener("input", function () {
            var b = Number(this.value);
            bearingVal.textContent = fmtBearing(b);
            if (map) map.setBearing(b);
        });

        document.getElementById("auto-rotate-btn").addEventListener("click", autoRotate);
        document.getElementById("save-btn").addEventListener("click", exportPNG);

        document.getElementById("poster-title").addEventListener("input", function () {
            posterTitle = this.value;
            if (statsData) renderStatsOverlay(statsData);
        });

        document.getElementById("stats-top-checkbox").addEventListener("change", function () {
            statsOnTop = this.checked;
            document.getElementById("stats-overlay").classList.toggle("stats-top", statsOnTop);
        });

        // <details> has no built-in click-away-to-close.
        document.addEventListener("click", function (e) {
            document.querySelectorAll("details.toolbar-popover[open]").forEach(function (d) {
                if (!d.contains(e.target)) d.removeAttribute("open");
            });
        });
    }

    function exportPNG() {
        var btn = document.getElementById("save-btn");
        btn.disabled = true;
        btn.textContent = "Rendering…";

        map.triggerRepaint();
        requestAnimationFrame(function () {
            captureAndDownload(btn);
        });
    }

    function captureAndDownload(btn) {
        function done() {
            btn.disabled = false;
            btn.textContent = "Save as PNG";
        }

        var bgImage = document.getElementById("bg-image");
        var frame   = document.getElementById("poster-frame");
        var dpr     = window.devicePixelRatio || 1;
        var exportW = Math.round(frame.clientWidth  * dpr);
        var exportH = Math.round(frame.clientHeight * dpr);

        var canvas = document.createElement("canvas");
        canvas.width  = exportW;
        canvas.height = exportH;
        var ctx = canvas.getContext("2d");

        // Layer 1: background photo.
        ctx.drawImage(bgImage, 0, 0, exportW, exportH);

        var opacity = document.getElementById("opacity-slider").value / 100;

        function afterMap() {
            // Layer 3: stats panel.
            if (statsData && statsData.length > 0) {
                drawStatsCanvas(ctx, aggregateStats(statsData), exportW, exportH);
            }

            // Layer 4: GPX tracks and mini-charts at full opacity, on top of the stats
            // panel — matches the live preview's z-order (#track-canvas above #stats-overlay).
            // trackCanvas is a plain 2D canvas — no WebGL / cross-origin issue.
            ctx.globalAlpha = 1;
            ctx.drawImage(trackCanvas, 0, 0, exportW, exportH);

            var link = document.createElement("a");
            link.download = "gpx-poster.png";
            link.href = canvas.toDataURL("image/png");
            link.click();
            done();
        }

        // Layer 2: map (roads/labels) at slider opacity.
        // toDataURL forces a synchronous GPU read-back before we copy.
        try {
            var mapDataUrl = map.getCanvas().toDataURL("image/png");
            var mapImg = new Image();
            mapImg.onload = function () {
                ctx.globalAlpha = opacity;
                ctx.drawImage(mapImg, 0, 0, exportW, exportH);
                ctx.globalAlpha = 1;
                afterMap();
            };
            mapImg.onerror = afterMap;
            mapImg.src = mapDataUrl;
        } catch (e) {
            console.warn("Map canvas read-back failed:", e);
            afterMap();
        }
    }

    function drawStatsCanvas(ctx, agg, w, h) {
        var frame   = document.getElementById("poster-frame");
        var overlay = document.getElementById("stats-overlay");
        var scale   = w / frame.clientWidth;

        var overlayH = Math.round(overlay.getBoundingClientRect().height * scale);
        var panelY   = statsOnTop ? 0 : h - overlayH;
        var cs   = window.getComputedStyle(overlay);
        var padL = Math.round(parseFloat(cs.paddingLeft) * scale);
        var padT = Math.round(parseFloat(cs.paddingTop)  * scale);

        var nameSz = Math.round(parseFloat(window.getComputedStyle(overlay.querySelector(".stats-name")).fontSize) * scale);
        var dateSz = Math.round(parseFloat(window.getComputedStyle(overlay.querySelector(".stats-date")).fontSize) * scale);

        // Guard: .stat-value / .stat-label may be absent when all stats are hidden.
        var valSz = 0, lblSz = 0;
        var firstVal = overlay.querySelector(".stat-value");
        var firstLbl = overlay.querySelector(".stat-label");
        if (firstVal) valSz = Math.round(parseFloat(window.getComputedStyle(firstVal).fontSize) * scale);
        if (firstLbl) lblSz = Math.round(parseFloat(window.getComputedStyle(firstLbl).fontSize) * scale);

        ctx.fillStyle = "rgba(0,0,0,0.72)";
        ctx.fillRect(0, panelY, w, overlayH);

        var stripeH = Math.max(2, Math.round(3 * scale));
        ctx.fillStyle = "#f97316";
        ctx.fillRect(0, statsOnTop ? panelY + overlayH - stripeH : panelY, w, stripeH);

        ctx.textAlign    = "left";
        ctx.textBaseline = "top";
        var y = panelY + padT;

        ctx.fillStyle = "#ffffff";
        ctx.font = "bold " + nameSz + "px system-ui,-apple-system,sans-serif";
        ctx.fillText(posterTitle || agg.name || "GPX Track", padL, y);
        y += Math.round(nameSz * 1.3);

        if (agg.date) {
            ctx.fillStyle = "rgba(255,255,255,0.55)";
            ctx.font = dateSz + "px system-ui,-apple-system,sans-serif";
            ctx.fillText(agg.date, padL, y);
            y += Math.round(dateSz * 1.7);
        }

        var items = STAT_DEFS
            .filter(function (d) { return enabledStats[d.key]; })
            .map(function (d) { return { value: overrides[d.key] || d.value(agg), label: d.gridLabel }; });

        if (items.length > 0 && valSz > 0) {
            var colW = w / items.length;
            items.forEach(function (item, i) {
                var cx = Math.round(colW * i + colW / 2);

                ctx.fillStyle = "#ffffff";
                ctx.font = "bold " + valSz + "px system-ui,-apple-system,sans-serif";
                ctx.textAlign = "center";
                ctx.fillText(item.value, cx, y);

                ctx.fillStyle = "rgba(255,255,255,0.5)";
                ctx.font = lblSz + "px system-ui,-apple-system,sans-serif";
                ctx.fillText(item.label, cx, y + Math.round(valSz * 1.3));
            });
        }

        ctx.textAlign    = "left";
        ctx.textBaseline = "alphabetic";
    }

    init();
}());
