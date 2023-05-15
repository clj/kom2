from http.server import BaseHTTPRequestHandler
from http.server import HTTPServer
import json
from urllib.parse import parse_qs


class MockHTTPRequestHandler(BaseHTTPRequestHandler):
    def do_GET(self):
        data = None
        if content_length := self.headers['Content-Length']:
            data = self.rfile.read(int(content_length))

        if '?' in self.path:
            path, args = self.path.split('?')
        else:
            path = self.path
            args = ""
        fn_name = path.strip('/').replace('/', '_')
        fn = getattr(self, fn_name)
        if not fn:
            self.send_response(404)
            self.end_headers()

            return

        self.send_response(200)
        self.send_header('Content-type', 'application/json')
        self.end_headers()

        self.wfile.write(json.dumps(fn(data, parse_qs(args))).encode('utf8'))

    def api_user_token(self, _, __):
        return {
            "token": "0123456789012345678901234567890123456789"
        }

    def api_part_category(self, _, __):
        return json.loads("""
            [
                {
                    "pk": 6,
                    "name": "Capacitors",
                    "description": "",
                    "default_location": null,
                    "default_keywords": null,
                    "level": 0,
                    "parent": null,
                    "part_count": 9,
                    "pathstring": "Capacitors",
                    "starred": false,
                    "url": "/part/category/6/",
                    "structural": false,
                    "icon": ""
                },
                {
                    "pk": 8,
                    "name": "Aluminium",
                    "description": "",
                    "default_location": null,
                    "default_keywords": null,
                    "level": 1,
                    "parent": 6,
                    "part_count": 1,
                    "pathstring": "Capacitors/Aluminium",
                    "starred": false,
                    "url": "/part/category/8/",
                    "structural": false,
                    "icon": ""
                },
                {
                    "pk": 59,
                    "name": "Resistors",
                    "description": "",
                    "default_location": null,
                    "default_keywords": null,
                    "level": 0,
                    "parent": null,
                    "part_count": 6,
                    "pathstring": "Resistors",
                    "starred": false,
                    "url": "/part/category/59/",
                    "structural": false,
                    "icon": ""
                },
                {
                    "pk": 63,
                    "name": "NTC",
                    "description": "",
                    "default_location": null,
                    "default_keywords": null,
                    "level": 1,
                    "parent": 59,
                    "part_count": 0,
                    "pathstring": "Resistors/NTC",
                    "starred": false,
                    "url": "/part/category/63/",
                    "structural": false,
                    "icon": ""
                }
            ]
        """)

    def api_part(self, _, __):
        parts = json.loads("""
        [
            {
                    "active": true,
                    "allocated_to_build_orders": 0.0,
                    "allocated_to_sales_orders": 0.0,
                    "assembly": false,
                    "barcode_hash": "",
                    "category": 60,
                    "component": true,
                    "default_expiry": 0,
                    "default_location": null,
                    "default_supplier": null,
                    "description": "SMD-Resistor, 0805, 0 Ohm, 0%",
                    "full_name": "RES-000014-00 | 0R resistor 0% SMD 0805",
                    "image": null,
                    "in_stock": 100.0,
                    "ordering": 0.0,
                    "building": 0.0,
                    "IPN": "RES-000014-00",
                    "is_template": false,
                    "keywords": "SMD-Resistor 0805 0Ohm 125mW 0% 0R",
                    "last_stocktake": null,
                    "link": "",
                    "minimum_stock": 0,
                    "name": "0R resistor 0% SMD 0805",
                    "notes": null,
                    "pk": 16,
                    "purchaseable": true,
                    "revision": "",
                    "salable": false,
                    "starred": false,
                    "stock_item_count": 1,
                    "suppliers": 1,
                    "thumbnail": "/static/img/blank_image.thumbnail.png",
                    "total_in_stock": 100.0,
                    "trackable": false,
                    "unallocated_stock": 100.0,
                    "units": "",
                    "variant_of": null,
                    "variant_stock": 0.0,
                    "virtual": false,
                    "pricing_min": null,
                    "pricing_max": null,
                    "responsible": null,
                    "copy_category_parameters": true
                },
                {
                    "active": true,
                    "allocated_to_build_orders": 0.0,
                    "allocated_to_sales_orders": 0.0,
                    "assembly": false,
                    "barcode_hash": "",
                    "category": 60,
                    "component": true,
                    "default_expiry": 0,
                    "default_location": null,
                    "default_supplier": null,
                    "description": "TRU COMPONENTS TC-0805S8F1003T5E203 Cermet resistor 100 kΩ SMD 0805 0.125 W 1 % 100 ppm/°C 1 pc(s) Tape cut",
                    "full_name": "RES-000037-00 | 100 kΩ SMD 0805 0.125 W 1 %",
                    "image": null,
                    "in_stock": 100.0,
                    "ordering": 0.0,
                    "building": 0.0,
                    "IPN": "RES-000037-00",
                    "is_template": false,
                    "keywords": "100k 0.125W SMD resistor",
                    "last_stocktake": null,
                    "link": "",
                    "minimum_stock": 0,
                    "name": "100 kΩ SMD 0805 0.125 W 1 %",
                    "notes": null,
                    "pk": 37,
                    "purchaseable": true,
                    "revision": "",
                    "salable": false,
                    "starred": false,
                    "stock_item_count": 1,
                    "suppliers": 1,
                    "thumbnail": "/static/img/blank_image.thumbnail.png",
                    "total_in_stock": 100.0,
                    "trackable": false,
                    "unallocated_stock": 100.0,
                    "units": "",
                    "variant_of": null,
                    "variant_stock": 0.0,
                    "virtual": false,
                    "pricing_min": null,
                    "pricing_max": null,
                    "responsible": null,
                    "copy_category_parameters": true
                },
                {
                    "active": true,
                    "allocated_to_build_orders": 0.0,
                    "allocated_to_sales_orders": 0.0,
                    "assembly": false,
                    "barcode_hash": "",
                    "category": 7,
                    "component": true,
                    "default_expiry": 0,
                    "default_location": null,
                    "default_supplier": null,
                    "description": "MLCC 100nF 50V 10% 0805 X7R Capacitor",
                    "full_name": "CAP-000015-00 | 100nF Ceramic Capacitor 50V 10% 0805",
                    "image": null,
                    "in_stock": 200.0,
                    "ordering": 0.0,
                    "building": 0.0,
                    "IPN": "CAP-000015-00",
                    "is_template": false,
                    "keywords": "MLCC 100nF 50V 10% 0805 X7R Capacitor",
                    "last_stocktake": null,
                    "link": "",
                    "minimum_stock": 0,
                    "name": "100nF Ceramic Capacitor 50V 10% 0805",
                    "notes": null,
                    "pk": 18,
                    "purchaseable": true,
                    "revision": "",
                    "salable": false,
                    "starred": false,
                    "stock_item_count": 1,
                    "suppliers": 1,
                    "thumbnail": "/static/img/blank_image.thumbnail.png",
                    "total_in_stock": 200.0,
                    "trackable": false,
                    "unallocated_stock": 200.0,
                    "units": "",
                    "variant_of": null,
                    "variant_stock": 0.0,
                    "virtual": false,
                    "pricing_min": null,
                    "pricing_max": null,
                    "responsible": null,
                    "copy_category_parameters": true
                },
                {
                    "active": true,
                    "allocated_to_build_orders": 0.0,
                    "allocated_to_sales_orders": 0.0,
                    "assembly": false,
                    "barcode_hash": "",
                    "category": 8,
                    "component": true,
                    "default_expiry": 0,
                    "default_location": null,
                    "default_supplier": null,
                    "description": "-55℃~+105℃ 3000hrs@105℃ 100uF 7.7mm 35V 6.3mm 230mA@100kHz 600mΩ@100kHz ±20% SMD,6.3x7.7mm  Aluminum Electrolytic Capacitors - SMD ROHS",
                    "full_name": "CAP-000030-00 | 100uF  35V  230mA@100kHz 600mΩ@100kHz ±20% SMD,6.3x7.7mm  Aluminum Electrolytic Capacitors | A",
                    "image": "/media/part_images/30_thumbnail.jpeg",
                    "in_stock": 30.0,
                    "ordering": 0.0,
                    "building": 0.0,
                    "IPN": "CAP-000030-00",
                    "is_template": false,
                    "keywords": "-55℃~+105℃ 3000hrs@105℃ 100uF 7.7mm 35V 6.3mm 230mA@100kHz 600mΩ@100kHz ±20% SMD,6.3x7.7mm  Aluminum Electrolytic Capacitors - SMD ROHS",
                    "last_stocktake": null,
                    "link": null,
                    "minimum_stock": 0,
                    "name": "100uF  35V  230mA@100kHz 600mΩ@100kHz ±20% SMD,6.3x7.7mm  Aluminum Electrolytic Capacitors",
                    "notes": null,
                    "pk": 30,
                    "purchaseable": true,
                    "revision": "A",
                    "salable": false,
                    "starred": false,
                    "stock_item_count": 1,
                    "suppliers": 1,
                    "thumbnail": "/media/part_images/30_thumbnail.thumbnail.jpeg",
                    "total_in_stock": 30.0,
                    "trackable": false,
                    "unallocated_stock": 30.0,
                    "units": "",
                    "variant_of": null,
                    "variant_stock": 0.0,
                    "virtual": false,
                    "pricing_min": null,
                    "pricing_max": null,
                    "responsible": null,
                    "copy_category_parameters": true
                }
            ]
        """)

        return parts

    def api_part_30(self, _, __):
        for part in self.api_part(_, __):
            if part['pk'] == 30:
                return part

    def api_part_30_metadata(self, _, __):
        return json.loads("""
            {
                "metadata": {
                    "kicad": {
                        "symbols": "symbol",
                        "footprints": "footprint"
                    }
                }
            }
        """)

    def api_part_parameter(self, _, __):
        return json.loads("""
            [
                {
                    "pk": 1,
                    "part": 1,
                    "template": 3,
                    "template_detail": {
                        "pk": 3,
                        "name": "Breakdown Voltage",
                        "units": "V",
                        "description": ""
                    },
                    "data": "6V"
                },
                {
                    "pk": 2,
                    "part": 1,
                    "template": 5,
                    "template_detail": {
                        "pk": 5,
                        "name": "Clamping Voltage",
                        "units": "V",
                        "description": ""
                    },
                    "data": "17V"
                }
            ]
        """)


if __name__ == "__main__":
    server_address = ('', 45454)
    httpd = HTTPServer(server_address, MockHTTPRequestHandler)
    httpd.serve_forever()
