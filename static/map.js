/**
 * @typedef Params
 * @property {String} tiles
 * @property {Array<Number,String>} files
 */

/**
 *
 * @param {Params} params
 */
function loadTracks(params) {
    "use strict";

    /**
     * @typedef ctxCb
     * @property {String} strokeStyle - [MDN Reference](https://developer.mozilla.org/docs/Web/API/CanvasRenderingContext2D/strokeStyle).
     * @property {Number} lineWidth - [MDN Reference](https://developer.mozilla.org/docs/Web/API/CanvasRenderingContext2D/lineWidth).
     */

    const polycolorRenderer = L.Canvas.extend({
        _updatePoly: function (layer) {
            if (!this._drawing) return;

            let i, j, len2, point, prevPoint;

            if (!layer._parts.length) return;
            console.log('layer:',layer, "this", this);

            this._layers[layer._leaflet_id] = layer;
            const ctx = this._ctx;

            if (ctx.setLineDash) {
                ctx.setLineDash(layer.options && layer.options._dashArray || []);
            }
            ctx.globalAlpha = layer.options.opacity;
            ctx.lineCap = layer.options.lineCap;
            ctx.lineJoin = layer.options.lineJoin;

            var dynCtx = null;
            if (typeof layer.options.dynamicCtx === 'function') {
                dynCtx = layer.options.dynamicCtx
            }

            for (i = 0; i < layer._parts.length; i++) {
                for (j = 0, len2 = layer._parts[i].length - 1; j < len2; j++) {
                    point = layer._parts[i][j + 1];
                    prevPoint = layer._parts[i][j];

                    ctx.beginPath();
                    ctx.moveTo(prevPoint.x, prevPoint.y);
                    ctx.lineTo(point.x, point.y);

                    ctx.strokeStyle = layer.options.color;
                    ctx.lineWidth = layer.options.weight;

                    if (dynCtx) {
                        dynCtx(point, ctx);
                    }

                    ctx.stroke();
                    ctx.closePath();
                }
            }
        },
    });

    var overlayMaps = {};

    var map = L.map('map', {
        fullscreenControl: true,
    })

    L.tileLayer(params.tiles, {}).addTo(map);

    function getDarkColor() {
        var color = '#';
        for (var i = 0; i < 6; i++) {
            color += Math.floor(Math.random() * 12).toString(16);
        }
        return color;
    }

    function getDarkColor2(seed) {
        var color = '#';
        for (var i = 0; i < 6; i++) {
            color += Math.floor((seed + (i*11)) % 14).toString(16);
        }
        return color;
    }

    // distance returns 2D distance between two points in meters.
    function distance(lat1, lon1, lat2, lon2) {
        var dLat = toRad * (lat1 - lat2)
        var dLon = toRad * (lon1 - lon2)

        var a = Math.pow(Math.sin(dLat / 2), 2) + Math.pow(Math.sin(dLon / 2), 2) * Math.cos(toRad * lat1) * Math.cos(toRad * lat2)
        var c = 2 * Math.atan2(Math.sqrt(a), Math.sqrt(1 - a))

        return 6371000 * c
    }

    function gpxPopupHandler(name) {
        return function (e) {
            console.log(e)

            var popup = e.popup;

            var feat = e.layer.feature
            console.log("feat", feat)

            var res = name + "<br />"
            if (feat.properties.desc) {
                res += feat.properties.desc + "<br />"
            }

            if (feat.properties.name) {
                res += feat.properties.name + "<br />"
            }

            if (feat.geometry.type === "Point") {
                res += feat.geometry.coordinates[1].toFixed(8) + ", " + feat.geometry.coordinates[0].toFixed(8)
                res += '<br /><a class="google-maps" href="https://maps.google.com/maps?q=loc:' +
                    feat.geometry.coordinates[1].toFixed(8) + ',' + feat.geometry.coordinates[0].toFixed(8) + '">google maps</a>'
                res += '<br /><a class="apple-maps" href="https://maps.apple.com/?ll=' +
                    feat.geometry.coordinates[1].toFixed(8) + ',' + feat.geometry.coordinates[0].toFixed(8) + '">apple maps</a>'

                popup.setContent(res);
                return
            }

            var points = e.layer._latlngs || []

            var lat = popup.getLatLng().lat
            var lon = popup.getLatLng().lng

            res += lat.toFixed(8) + ", " + lon.toFixed(8)

            if (typeof e.layer.feature.properties.coordTimes !== 'undefined') {
                var shortest = null
                var ptIdx = []
                for (var i = 0; i < points.length; i++) {
                    var pt = points[i]

                    if (pt instanceof Array) {
                        for (var j = 0; j < pt.length; j++) {
                            var pt2 = pt[j]

                            var dist = distance(lat, lon, pt2.lat, pt2.lng)

                            if (shortest === null || shortest > dist) {
                                shortest = dist
                                ptIdx = [i, j]
                            }
                        }

                        continue
                    }

                    var dist = distance(lat, lon, pt.lat, pt.lng)

                    if (shortest === null || shortest > dist) {
                        shortest = dist
                        ptIdx = [i]
                    }
                }

                var ts

                if (ptIdx.length == 1) {
                    ts = e.layer.feature.properties.coordTimes[ptIdx[0]]
                }

                if (ptIdx.length == 2) {
                    ts = e.layer.feature.properties.coordTimes[ptIdx[0]][ptIdx[1]]
                }

                res +=
                    '<br />' + ts + " (" + Math.round(shortest) + "m away)"
            }

            res += '<br /><a class="google-maps" href="https://maps.google.com/maps?q=loc:' +
                lat.toFixed(8) + ',' + lon.toFixed(8) + '">google maps</a>'
            res += '<br /><a class="apple-maps" href="https://maps.apple.com/?ll=' +
                lat.toFixed(8) + ',' + lon.toFixed(8) + '">apple maps</a>'

            popup.setContent(res);
        }
    }

    var bounds = null
    var toRad = Math.PI / 180

    // GPX rendering.
    for (var i = 0; i < params.files.length; i++) {
        var customLayer = L.geoJson(null, {
            // style: function () {
            //     return {color: getDarkColor()};
            // },
            style: function (feature) {

                return {
                    // noClip: true,
                    // smoothFactor: 0,
                    color: getDarkColor(),
                    renderer: new polycolorRenderer(),
                    // dynamicCtx: function (point, ctx) {
                    //     ctx.strokeStyle = getDarkColor2(100*point.x+point.y)
                    //     ctx.lineWidth = (100*point.x+point.y) % 15 + 5
                    // }
                }
            }
        });

        var name = params.files[i];
        var gpxLayer = omnivore.gpx('/track/' + i + '.gpx', null, customLayer)
            .on('ready', function (e) {
                var b = e.target.getBounds()

                if (bounds === null) {
                    bounds = b
                } else {
                    bounds.extend(b)
                }

                map.fitBounds(bounds);
            });
        gpxLayer
            .bindPopup(function () {
                return name
            })
            .addTo(map);

        gpxLayer.on('popupopen', gpxPopupHandler(name));

        overlayMaps[name] = gpxLayer
    }

    L.control.layers({}, overlayMaps).addTo(map);
    L.control.locate({}).addTo(map)
}


