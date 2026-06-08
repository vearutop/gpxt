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

    map.on("load", function () {
        params.files.forEach(function (name, i) {
            const sourceId = "track-" + i;
            const layerId = "track-line-" + i;

            fetch("/track/" + i + ".geojson")
                .then(function (response) { return response.json(); })
                .then(function (geojson) {
                    geojson.features.forEach(function (feature) {
                        feature.properties = feature.properties || {};
                        feature.properties.trackName = name;
                    });

                    map.addSource(sourceId, {
                        type: "geojson",
                        data: geojson
                    });

                    map.addLayer({
                        id: layerId,
                        type: "line",
                        source: sourceId,
                        paint: {
                            "line-color": colorForIndex(i),
                            "line-width": 4
                        }
                    });

                    geojson.features.forEach(function (feature) {
                        feature.geometry.coordinates.forEach(function (coord) {
                            bounds.extend(coord);
                        });
                    });

                    map.on("click", layerId, function (event) {
                        const props = event.features[0].properties || {};
                        const popup = new maplibregl.Popup({ closeButton: true });
                        popup
                            .setLngLat(event.lngLat)
                            .setHTML("<b>" + (props.trackName || name) + "</b><br/>" + (props.source || ""))
                            .addTo(map);
                    });

                    map.on("mouseenter", layerId, function () {
                        map.getCanvas().style.cursor = "pointer";
                    });

                    map.on("mouseleave", layerId, function () {
                        map.getCanvas().style.cursor = "";
                    });

                    if (!bounds.isEmpty()) {
                        map.fitBounds(bounds, { padding: 32, duration: 0 });
                    }
                });
        });
    });
}
