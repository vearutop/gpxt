function loadTracks(params) {
    "use strict";

    const map = new maplibregl.Map({
        container: "map",
        style: params.styleURL,
        center: [0, 0],
        zoom: 1,
        attributionControl: true
    });

    map.addControl(new maplibregl.NavigationControl());

    const bounds = new maplibregl.LngLatBounds();

    function colorForIndex(index) {
        const palette = ["#f97316", "#22c55e", "#06b6d4", "#eab308", "#ef4444", "#8b5cf6", "#14b8a6", "#f43f5e"];
        return palette[index % palette.length];
    }

    function setTrackVisibility(index, visible) {
        const layerIds = ["track-line-" + index, "track-point-" + index];
        layerIds.forEach(function (layerId) {
            if (!map.getLayer(layerId)) {
                return;
            }

            map.setLayoutProperty(layerId, "visibility", visible ? "visible" : "none");
        });
    }

    function addGeoJsonLayers(name, i, geojson) {
        const sourceId = "track-" + i;
        const lineLayerId = "track-line-" + i;
        const pointLayerId = "track-point-" + i;

        map.addSource(sourceId, {
            type: "geojson",
            data: geojson
        });

        const hasLines = geojson.features.some(function (feature) {
            return feature.geometry && feature.geometry.type === "LineString";
        });
        const hasPoints = geojson.features.some(function (feature) {
            return feature.geometry && feature.geometry.type === "Point";
        });

        if (hasLines) {
            map.addLayer({
                id: lineLayerId,
                type: "line",
                source: sourceId,
                filter: ["==", ["geometry-type"], "LineString"],
                paint: {
                    "line-color": colorForIndex(i),
                    "line-width": 4
                }
            });
        }

        if (hasPoints) {
            map.addLayer({
                id: pointLayerId,
                type: "circle",
                source: sourceId,
                filter: ["==", ["geometry-type"], "Point"],
                paint: {
                    "circle-color": colorForIndex(i),
                    "circle-radius": 6,
                    "circle-stroke-width": 2,
                    "circle-stroke-color": "#ffffff"
                }
            });
        }

        geojson.features.forEach(function (feature) {
            if (feature.geometry.type === "LineString") {
                feature.geometry.coordinates.forEach(function (coord) {
                    bounds.extend(coord);
                });
            } else if (feature.geometry.type === "Point") {
                bounds.extend(feature.geometry.coordinates);
            }
        });

        if (hasLines) {
            map.on("click", lineLayerId, function (event) {
                const props = event.features[0].properties || {};
                const popup = new maplibregl.Popup({ closeButton: true });
                popup
                    .setLngLat(event.lngLat)
                    .setHTML("<b>" + (props.trackName || name) + "</b><br/>" + (props.source || ""))
                    .addTo(map);
            });

            map.on("mouseenter", lineLayerId, function () {
                map.getCanvas().style.cursor = "pointer";
            });

            map.on("mouseleave", lineLayerId, function () {
                map.getCanvas().style.cursor = "";
            });
        }

        if (hasPoints) {
            map.on("click", pointLayerId, function (event) {
                const props = event.features[0].properties || {};
                const popup = new maplibregl.Popup({ closeButton: true });
                popup
                    .setLngLat(event.lngLat)
                    .setHTML("<b>" + (props.name || name) + "</b><br/>" + (props.desc || "") + (props.time ? "<br/>" + props.time : ""))
                    .addTo(map);
            });

            map.on("mouseenter", pointLayerId, function () {
                map.getCanvas().style.cursor = "pointer";
            });

            map.on("mouseleave", pointLayerId, function () {
                map.getCanvas().style.cursor = "";
            });
        }

        setTrackVisibility(i, true);
    }

    function buildTrackControls() {
        const panel = document.createElement("div");
        panel.className = "track-panel";

        const title = document.createElement("div");
        title.className = "track-panel__title";
        title.textContent = "Tracks";
        panel.appendChild(title);

        params.files.forEach(function (name, i) {
            const row = document.createElement("label");
            row.className = "track-panel__item";

            const checkbox = document.createElement("input");
            checkbox.type = "checkbox";
            checkbox.checked = true;
            checkbox.addEventListener("change", function () {
                setTrackVisibility(i, checkbox.checked);
            });

            const swatch = document.createElement("span");
            swatch.className = "track-panel__swatch";
            swatch.style.backgroundColor = colorForIndex(i);

            const text = document.createElement("span");
            text.className = "track-panel__label";
            text.textContent = name;

            row.appendChild(checkbox);
            row.appendChild(swatch);
            row.appendChild(text);
            panel.appendChild(row);
        });

        map.getContainer().appendChild(panel);
    }

    map.on("load", function () {
        buildTrackControls();
        params.files.forEach(function (name, i) {
            fetch("/track/" + i + ".geojson")
                .then(function (response) { return response.json(); })
                .then(function (geojson) {
                    geojson.features.forEach(function (feature) {
                        feature.properties = feature.properties || {};
                        feature.properties.trackName = name;
                    });

                    addGeoJsonLayers(name, i, geojson);

                    if (!bounds.isEmpty()) {
                        map.fitBounds(bounds, { padding: 32, duration: 0 });
                    }
                });
        });
    });
}
