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
            map.resize();
            loadTracks();
        });

        var opacity = document.getElementById("opacity-slider").value / 100;
        map.getContainer().style.opacity = String(opacity);
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

        setTimeout(function () {
            try {
                var bgImage = document.getElementById("bg-image");
                var exportW = 1080;
                var exportH = Math.round(exportW * bgImage.naturalHeight / bgImage.naturalWidth);

                var canvas = document.createElement("canvas");
                canvas.width = exportW;
                canvas.height = exportH;
                var ctx = canvas.getContext("2d");

                // Layer 1: background photo.
                ctx.drawImage(bgImage, 0, 0, exportW, exportH);

                // Layer 2: map at configured opacity.
                var opacity = document.getElementById("opacity-slider").value / 100;
                ctx.globalAlpha = opacity;
                ctx.drawImage(map.getCanvas(), 0, 0, exportW, exportH);
                ctx.globalAlpha = 1;

                // Layer 3: stats panel drawn directly on canvas.
                if (statsData && statsData.length > 0) {
                    drawStatsCanvas(ctx, aggregateStats(statsData), exportW, exportH);
                }

                var link = document.createElement("a");
                link.download = "gpx-poster.png";
                link.href = canvas.toDataURL("image/png");
                link.click();
            } catch (e) {
                alert("Export failed: " + e.message);
            } finally {
                btn.disabled = false;
                btn.textContent = "Save as PNG";
            }
        }, 0);
    }

    function drawStatsCanvas(ctx, agg, w, h) {
        var panelH = Math.round(h * 0.21);
        var panelY = h - panelH;
        var pad = Math.round(w * 0.046);

        // Background.
        ctx.fillStyle = "rgba(0,0,0,0.72)";
        ctx.fillRect(0, panelY, w, panelH);

        // Accent line.
        ctx.fillStyle = "#f97316";
        ctx.fillRect(0, panelY, w, Math.round(h * 0.003));

        // Track name.
        var nameSz = Math.round(w * 0.042);
        ctx.fillStyle = "#ffffff";
        ctx.font = "bold " + nameSz + "px system-ui,-apple-system,sans-serif";
        ctx.textAlign = "left";
        ctx.textBaseline = "alphabetic";
        ctx.fillText(agg.name || "GPX Track", pad, panelY + Math.round(panelH * 0.36));

        // Date.
        if (agg.date) {
            var dateSz = Math.round(w * 0.026);
            ctx.fillStyle = "rgba(255,255,255,0.55)";
            ctx.font = dateSz + "px system-ui,-apple-system,sans-serif";
            ctx.fillText(agg.date, pad, panelY + Math.round(panelH * 0.57));
        }

        // Four stat columns.
        var items = [
            { value: agg.distanceKm.toFixed(1) + " km", label: "Distance" },
            { value: agg.movingTime, label: "Moving Time" },
            { value: agg.avgSpeedKmh.toFixed(1) + " km/h", label: "Avg Speed" },
            { value: "↑ " + Math.round(agg.uphillM) + " m", label: "Elevation" },
        ];

        var valSz = Math.round(w * 0.036);
        var lblSz = Math.round(w * 0.024);
        var colW = w / items.length;

        items.forEach(function (item, i) {
            var cx = Math.round(colW * i + colW / 2);

            ctx.fillStyle = "#ffffff";
            ctx.font = "bold " + valSz + "px system-ui,-apple-system,sans-serif";
            ctx.textAlign = "center";
            ctx.textBaseline = "alphabetic";
            ctx.fillText(item.value, cx, panelY + Math.round(panelH * 0.72));

            ctx.fillStyle = "rgba(255,255,255,0.5)";
            ctx.font = lblSz + "px system-ui,-apple-system,sans-serif";
            ctx.fillText(item.label, cx, panelY + Math.round(panelH * 0.9));
        });

        ctx.textAlign = "left";
        ctx.textBaseline = "alphabetic";
    }

    init();
}());
