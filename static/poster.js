(function () {
    "use strict";

    var config = window.POSTER_CONFIG;
    var map = null;
    var statsData = null;
    var tracksLoaded = 0;

    function init() {
        loadBackgroundImage()
            .then(function () {
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
                // Force layout recalc before MapLibre init.
                frame.getBoundingClientRect();
                resolve();
            };
            img.onerror = function () { reject(new Error("Failed to load background image")); };
            img.src = "/image";
        });
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

        var opacity = document.getElementById("opacity-slider").value / 100;
        map.getContainer().style.opacity = String(opacity);
    }

    // Remove all filled/background/raster layers so the photo shows through cleanly.
    // Lines (roads, rivers) and symbol labels are kept.
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
        var palette = ["#f97316", "#22c55e", "#06b6d4", "#eab308", "#ef4444", "#8b5cf6"];

        config.files.forEach(function (name, i) {
            fetch("/track/" + i + ".geojson")
                .then(function (r) { return r.json(); })
                .then(function (geojson) {
                    map.addSource("track-" + i, { type: "geojson", data: geojson });

                    var hasLines = geojson.features.some(function (f) {
                        return f.geometry && f.geometry.type === "LineString";
                    });

                    if (hasLines) {
                        // Dark outline for contrast on any background.
                        map.addLayer({
                            id: "track-outline-" + i,
                            type: "line",
                            source: "track-" + i,
                            filter: ["==", ["geometry-type"], "LineString"],
                            paint: {
                                "line-color": "#000000",
                                "line-width": 8,
                                "line-opacity": 0.45,
                            },
                        });

                        map.addLayer({
                            id: "track-line-" + i,
                            type: "line",
                            source: "track-" + i,
                            filter: ["==", ["geometry-type"], "LineString"],
                            paint: {
                                "line-color": palette[i % palette.length],
                                "line-width": 5,
                            },
                        });
                    }

                    geojson.features.forEach(function (f) {
                        if (f.geometry.type === "LineString") {
                            f.geometry.coordinates.forEach(function (c) { bounds.extend(c); });
                        } else if (f.geometry.type === "Point") {
                            bounds.extend(f.geometry.coordinates);
                        }
                    });

                    if (!bounds.isEmpty()) {
                        map.fitBounds(bounds, { padding: 56, duration: 0 });
                    }

                    tracksLoaded++;
                    if (tracksLoaded === config.files.length) {
                        // Wait for tiles to render before enabling save.
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
            statItem("↑ " + Math.round(agg.uphillM) + " m", "Elevation") +
            "</div>";
    }

    function aggregateStats(stats) {
        var totalDist = 0, totalSecs = 0, totalUphill = 0;

        stats.forEach(function (s) {
            totalDist += s.distance_km;
            totalSecs += s.moving_time_secs;
            totalUphill += s.uphill_m;
        });

        var avgSpeedKmh = totalSecs > 0 ? (totalDist / totalSecs) * 3600 : 0;

        return {
            name: stats[0].name,
            date: stats[0].date,
            distanceKm: totalDist,
            movingTime: fmtDuration(totalSecs),
            avgSpeedKmh: avgSpeedKmh,
            uphillM: totalUphill,
        };
    }

    function fmtDuration(secs) {
        var h = Math.floor(secs / 3600);
        var m = Math.floor((secs % 3600) / 60);
        if (h > 0) return h + "h " + String(m).padStart(2, "0") + "m";
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
        var slider = document.getElementById("opacity-slider");
        var opacityVal = document.getElementById("opacity-value");

        slider.addEventListener("input", function () {
            opacityVal.textContent = slider.value + "%";
            if (map) map.getContainer().style.opacity = String(slider.value / 100);
        });

        document.getElementById("save-btn").addEventListener("click", exportPNG);
    }

    function exportPNG() {
        var btn = document.getElementById("save-btn");
        btn.disabled = true;
        btn.textContent = "Rendering…";

        // Trigger a fresh WebGL render, then capture on the next animation frame.
        // One rAF is enough: MapLibre renders into the canvas during this frame,
        // and toDataURL (used below) forces a synchronous GPU read-back.
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

        // Export at DPR × CSS frame size — matches what MapLibre rendered, no scaling.
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

        function finish() {
            // Layer 3: stats panel — sizes derived from live DOM, scaled to export.
            if (statsData && statsData.length > 0) {
                drawStatsCanvas(ctx, aggregateStats(statsData), exportW, exportH);
            }
            var link = document.createElement("a");
            link.download = "gpx-poster.png";
            link.href = canvas.toDataURL("image/png");
            link.click();
            done();
        }

        // Layer 2: map.
        // toDataURL serialises the WebGL buffer synchronously (GPU read-back),
        // then we load it as a plain Image so the 2D canvas drawImage path is used.
        // Direct drawImage(webglCanvas) is unreliable in Firefox when the WebGL
        // context renders with a transparent background (stripped fills).
        try {
            var mapDataUrl = map.getCanvas().toDataURL("image/png");
            var mapImg = new Image();
            mapImg.onload = function () {
                ctx.globalAlpha = opacity;
                ctx.drawImage(mapImg, 0, 0, exportW, exportH);
                ctx.globalAlpha = 1;
                finish();
            };
            mapImg.onerror = finish; // proceed without map layer on failure
            mapImg.src = mapDataUrl;
        } catch (e) {
            // Canvas tainted by cross-origin tiles — skip map layer.
            console.warn("Map canvas read-back failed:", e);
            finish();
        }
    }

    // Draw the stats panel onto the export canvas.
    // Font sizes and panel height are read from the live DOM and scaled up to
    // the export resolution so the PNG matches what the user sees on screen.
    function drawStatsCanvas(ctx, agg, w, h) {
        var frame = document.getElementById("poster-frame");
        var overlay = document.getElementById("stats-overlay");
        var scale = w / frame.clientWidth;

        // Mirror the DOM layout.
        var overlayH = Math.round(overlay.getBoundingClientRect().height * scale);
        var panelY = h - overlayH;
        var cs = window.getComputedStyle(overlay);
        var padL = Math.round(parseFloat(cs.paddingLeft) * scale);
        var padT = Math.round(parseFloat(cs.paddingTop) * scale);

        var nameSz = Math.round(parseFloat(window.getComputedStyle(overlay.querySelector(".stats-name")).fontSize) * scale);
        var dateSz = Math.round(parseFloat(window.getComputedStyle(overlay.querySelector(".stats-date")).fontSize) * scale);
        var valSz  = Math.round(parseFloat(window.getComputedStyle(overlay.querySelector(".stat-value")).fontSize) * scale);
        var lblSz  = Math.round(parseFloat(window.getComputedStyle(overlay.querySelector(".stat-label")).fontSize) * scale);

        // Panel background.
        ctx.fillStyle = "rgba(0,0,0,0.72)";
        ctx.fillRect(0, panelY, w, overlayH);

        // Accent line.
        ctx.fillStyle = "#f97316";
        ctx.fillRect(0, panelY, w, Math.max(2, Math.round(3 * scale)));

        ctx.textAlign = "left";
        ctx.textBaseline = "top";
        var y = panelY + padT;

        // Track name.
        ctx.fillStyle = "#ffffff";
        ctx.font = "bold " + nameSz + "px system-ui,-apple-system,sans-serif";
        ctx.fillText(agg.name || "GPX Track", padL, y);
        y += Math.round(nameSz * 1.3);

        // Date.
        if (agg.date) {
            ctx.fillStyle = "rgba(255,255,255,0.55)";
            ctx.font = dateSz + "px system-ui,-apple-system,sans-serif";
            ctx.fillText(agg.date, padL, y);
            y += Math.round(dateSz * 1.7);
        }

        // Four stat columns.
        var items = [
            { value: agg.distanceKm.toFixed(1) + " km", label: "Distance" },
            { value: agg.movingTime,                    label: "Moving Time" },
            { value: agg.avgSpeedKmh.toFixed(1) + " km/h", label: "Avg Speed" },
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

        ctx.textAlign = "left";
        ctx.textBaseline = "alphabetic";
    }

    init();
}());
