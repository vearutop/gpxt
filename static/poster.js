(function () {
    "use strict";

    var config = window.POSTER_CONFIG;
    var map = null;
    var statsData = null;
    var allGeoJson = [];   // indexed by file, filled as fetches complete
    var trackCanvas = null;
    var tc = null;         // 2D context of the track overlay canvas
    var palette = ["#f97316", "#22c55e", "#06b6d4", "#eab308", "#ef4444", "#8b5cf6"];

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
            stripMapFills();
            map.resize();
            loadTracks();
        });

        // Redraw the track overlay whenever MapLibre repaints.
        map.on("render", drawTracks);

        var opacity = document.getElementById("opacity-slider").value / 100;
        map.getContainer().style.opacity = String(opacity);
    }

    function stripMapFills() {
        map.getStyle().layers.forEach(function (layer) {
            switch (layer.type) {
            case "background":
                map.setPaintProperty(layer.id, "background-opacity", 0);
                break;
            case "fill":
                map.setPaintProperty(layer.id, "fill-opacity", 0);
                break;
            case "fill-extrusion":
                map.setPaintProperty(layer.id, "fill-extrusion-opacity", 0);
                break;
            case "raster":
                map.setPaintProperty(layer.id, "raster-opacity", 0);
                break;
            }
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
                            map.fitBounds(bounds, { padding: 56, duration: 0 });
                        }
                        map.once("idle", function () {
                            var mc = map.getCanvas();
                            document.getElementById("export-size").textContent =
                                mc.width + "×" + mc.height + " px";
                            document.getElementById("save-btn").disabled = false;
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
            var color = palette[i % palette.length];

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

        overlay.innerHTML =
            '<div class="stats-name">' + esc(agg.name) + "</div>" +
            '<div class="stats-date">' + esc(agg.date) + "</div>" +
            '<div class="stats-grid">' +
            statItem(agg.distanceKm.toFixed(1) + " km", "Distance") +
            statItem(agg.movingTime, "Moving Time") +
            statItem(agg.avgSpeedKmh.toFixed(1) + " km/h", "Avg Speed") +
            statItem("↑ " + Math.round(agg.uphillM) + " m", "Elevation") +
            "</div>";
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

    function setupControls() {
        var slider     = document.getElementById("opacity-slider");
        var opacityVal = document.getElementById("opacity-value");

        slider.addEventListener("input", function () {
            opacityVal.textContent = slider.value + "%";
            // Only the map container changes — track canvas is always at 100%.
            if (map) map.getContainer().style.opacity = String(slider.value / 100);
        });

        document.getElementById("save-btn").addEventListener("click", exportPNG);
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
        var valSz  = Math.round(parseFloat(window.getComputedStyle(overlay.querySelector(".stat-value")).fontSize) * scale);
        var lblSz  = Math.round(parseFloat(window.getComputedStyle(overlay.querySelector(".stat-label")).fontSize) * scale);

        ctx.fillStyle = "rgba(0,0,0,0.72)";
        ctx.fillRect(0, panelY, w, overlayH);

        ctx.fillStyle = "#f97316";
        ctx.fillRect(0, panelY, w, Math.max(2, Math.round(3 * scale)));

        ctx.textAlign    = "left";
        ctx.textBaseline = "top";
        var y = panelY + padT;

        ctx.fillStyle = "#ffffff";
        ctx.font = "bold " + nameSz + "px system-ui,-apple-system,sans-serif";
        ctx.fillText(agg.name || "GPX Track", padL, y);
        y += Math.round(nameSz * 1.3);

        if (agg.date) {
            ctx.fillStyle = "rgba(255,255,255,0.55)";
            ctx.font = dateSz + "px system-ui,-apple-system,sans-serif";
            ctx.fillText(agg.date, padL, y);
            y += Math.round(dateSz * 1.7);
        }

        var items = [
            { value: agg.distanceKm.toFixed(1) + " km",       label: "Distance" },
            { value: agg.movingTime,                           label: "Moving Time" },
            { value: agg.avgSpeedKmh.toFixed(1) + " km/h",    label: "Avg Speed" },
            { value: "↑ " + Math.round(agg.uphillM) + " m", label: "Elevation" },
        ];

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

        ctx.textAlign    = "left";
        ctx.textBaseline = "alphabetic";
    }

    init();
}());
