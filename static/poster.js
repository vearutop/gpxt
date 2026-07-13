(function () {
    "use strict";

    var config = window.POSTER_CONFIG;
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
    var enabledStats = { distance: true, movingTime: true, avgSpeed: true, elevation: true };

    function init() {
        loadBackgroundImage()
            .then(function () {
                initTrackCanvas();
                initMap();
                return fetch("/stats.json").then(function (r) { return r.json(); });
            })
            .then(function (stats) {
                statsData = stats;
                renderStatsOverlay(stats);
                setupControls();
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
            img.src = "/image";
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
            fetch("/track/" + i + ".geojson")
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

        var itemsHtml = [];
        if (enabledStats.distance)   itemsHtml.push(statItem(agg.distanceKm.toFixed(1) + " km", "Distance"));
        if (enabledStats.movingTime) itemsHtml.push(statItem(agg.movingTime, "Moving Time"));
        if (enabledStats.avgSpeed)   itemsHtml.push(statItem(agg.avgSpeedKmh.toFixed(1) + " km/h", "Avg Speed"));
        if (enabledStats.elevation)  itemsHtml.push(statItem("↑ " + Math.round(agg.uphillM) + " m", "Elevation"));

        overlay.innerHTML =
            '<div class="stats-name">' + esc(displayName) + "</div>" +
            '<div class="stats-date">' + esc(agg.date) + "</div>" +
            (itemsHtml.length
                ? '<div class="stats-grid" style="grid-template-columns:repeat(' + itemsHtml.length + ',1fr)">' +
                  itemsHtml.join("") + "</div>"
                : "");
    }

    function aggregateStats(stats) {
        var totalDist = 0, totalSecs = 0, totalUphill = 0;
        stats.forEach(function (s) {
            totalDist   += s.distance_km;
            totalSecs   += s.moving_time_secs;
            totalUphill += s.uphill_m;
        });
        return {
            name:        stats[0].name,
            date:        stats[0].date,
            distanceKm:  totalDist,
            movingTime:  fmtDuration(totalSecs),
            avgSpeedKmh: totalSecs > 0 ? (totalDist / totalSecs) * 3600 : 0,
            uphillM:     totalUphill,
        };
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
        document.getElementById("bearing-value").textContent = Math.round(bearing) + "°";
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

    function setupControls() {
        initTrackColorPickers();
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
            bearingVal.textContent = b + "°";
            if (map) map.setBearing(b);
        });
        // Re-fit track into frame once the drag is released.
        bearingSlider.addEventListener("change", function () {
            var b = Number(bearingSlider.value);
            if (map && trackBounds) map.fitBounds(trackBounds, { padding: 56, duration: 200, bearing: b });
        });

        document.getElementById("auto-rotate-btn").addEventListener("click", autoRotate);
        document.getElementById("save-btn").addEventListener("click", exportPNG);

        document.getElementById("poster-title").addEventListener("input", function () {
            posterTitle = this.value;
            if (statsData) renderStatsOverlay(statsData);
        });

        document.querySelectorAll("[data-stat]").forEach(function (checkbox) {
            checkbox.addEventListener("change", function () {
                enabledStats[this.dataset.stat] = this.checked;
                if (statsData) renderStatsOverlay(statsData);
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
            // Layer 3: GPX tracks at full opacity.
            // trackCanvas is a plain 2D canvas — no WebGL / cross-origin issue.
            ctx.globalAlpha = 1;
            ctx.drawImage(trackCanvas, 0, 0, exportW, exportH);

            // Layer 4: stats panel.
            if (statsData && statsData.length > 0) {
                drawStatsCanvas(ctx, aggregateStats(statsData), exportW, exportH);
            }

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
        var panelY   = h - overlayH;
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

        ctx.fillStyle = "#f97316";
        ctx.fillRect(0, panelY, w, Math.max(2, Math.round(3 * scale)));

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

        var items = [];
        if (enabledStats.distance)   items.push({ value: agg.distanceKm.toFixed(1) + " km",    label: "Distance" });
        if (enabledStats.movingTime) items.push({ value: agg.movingTime,                        label: "Moving Time" });
        if (enabledStats.avgSpeed)   items.push({ value: agg.avgSpeedKmh.toFixed(1) + " km/h", label: "Avg Speed" });
        if (enabledStats.elevation)  items.push({ value: "↑ " + Math.round(agg.uphillM) + " m", label: "Elevation" });

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
