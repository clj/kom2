import copy
import json
import platform
import sys

import jsonpointer
import pytest
import pypyodbc

from ..conftest import maybe_skip_windows


pytestmark = pytest.mark.skipif(
    maybe_skip_windows(), reason="Could not load kom2 driver on Windows"
)


@pytest.fixture
def token_resource(httpserver):
    httpserver.expect_request("/api/user/token").respond_with_json(
        {"token": "0123456789012345678901234567890123456789"}
    )


@pytest.fixture
def categories_resource(httpserver):
    httpserver.expect_request("/api/part/category/").respond_with_data(
        """
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
    """,
        content_type="application/json",
    )


parts = json.loads(
    """
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
            """
)


@pytest.fixture
def parts_resource(httpserver, request):
    modded_parts = copy.deepcopy(parts)
    marker = request.node.get_closest_marker("part_mods")
    if marker:
        for path, val in marker.args[0]:
            jsonpointer.set_pointer(modded_parts, path, val)
    httpserver.expect_request("/api/part/").respond_with_json(modded_parts)


@pytest.fixture
def part_resource(httpserver, request):
    modded_parts = copy.deepcopy(parts)
    marker = request.node.get_closest_marker("part_mods")
    if marker:
        for path, val in marker.args[0]:
            jsonpointer.set_pointer(modded_parts, path, val)
    for part, modded_part in zip(parts, modded_parts):
        httpserver.expect_request(f"/api/part/{part['pk']}/").respond_with_json(
            modded_part
        )


@pytest.fixture
def part_parameters_resource(httpserver):
    for part in parts:
        httpserver.expect_request(
            f"/api/part/parameter/", query_string=f"part={part['pk']}"
        ).respond_with_data(
            """
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
        """,
            content_type="application/json",
        )


def test_unconditional_select_invalid_table(
    httpserver, driver_name, token_resource, categories_resource
):
    server = httpserver.url_for("")
    cnxn = pypyodbc.connect(
        f"Driver={driver_name};server={server};username=asdf;password=asdf"
    )
    crsr = cnxn.cursor()
    crsr.prepare("SELECT * FROM Pizzas")
    # pypyodbc doesn't allow us to execute the prepares statements
    # unless we call the SQLExecute function directly
    ret = pypyodbc.SQLExecute(crsr.stmt_h)
    assert ret == pypyodbc.SQL_ERROR
    with pytest.raises(pypyodbc.Error) as exception:
        # Because SQLExecute was updated directly, also call:
        pypyodbc.check_success(crsr, ret)
    assert "Unable to fetch parts" in exception.value.args[1]
    assert "Category does not exist" in exception.value.args[1]


def test_unconditional_select_resource_error(
    httpserver, driver_name, token_resource, categories_resource
):
    server = httpserver.url_for("")
    cnxn = pypyodbc.connect(
        f"Driver={driver_name};server={server};username=asdf;password=asdf"
    )
    crsr = cnxn.cursor()
    crsr.prepare("SELECT * FROM Resistors")
    # pypyodbc doesn't allow us to execute the prepares statements
    # unless we call the SQLExecute function directly
    ret = pypyodbc.SQLExecute(crsr.stmt_h)
    assert ret == pypyodbc.SQL_ERROR
    with pytest.raises(pypyodbc.Error) as exception:
        # Because SQLExecute was updated directly, also call:
        pypyodbc.check_success(crsr, ret)
    assert "Unable to fetch parts" in exception.value.args[1]


@pytest.mark.parametrize(
    "query",
    [
        "SELECT * FROM Resistors",
        "SELECT IPN FROM Resistors",
        'SELECT "IPN" FROM Resistors',
        "SELECT IPN, pk FROM Resistors",
        'SELECT "IPN", "pk" FROM Resistors',
        "SELECT IPN,pk FROM Resistors",
        'SELECT "IPN","pk" FROM Resistors',
    ],
)
def test_unconditional_select(
    httpserver, driver_name, token_resource, categories_resource, parts_resource, query
):
    # TODO: check the category in query string for parts request
    server = httpserver.url_for("")
    cnxn = pypyodbc.connect(
        f"Driver={driver_name};server={server};username=asdf;password=asdf"
    )
    crsr = cnxn.cursor()
    crsr.prepare(query)
    # pypyodbc doesn't allow us to execute the prepares statements
    # unless we call the SQLExecute function directly
    ret = pypyodbc.SQLExecute(crsr.stmt_h)
    if ret != pypyodbc.SQL_SUCCESS:
        pypyodbc.check_success(crsr, ret)
    assert (
        ret == pypyodbc.SQL_SUCCESS
    )  # redundant, but what we actually are trying to do
    # Because SQLExecute was updated directly, also call:
    pypyodbc.check_success(crsr, ret)
    crsr._NumOfRows()
    crsr._UpdateDesc()

    results = crsr.fetchall()
    assert len(results) == 4


def test_conditional_select_invalid_condition_column(
    httpserver, driver_name, token_resource, categories_resource
):
    server = httpserver.url_for("")
    cnxn = pypyodbc.connect(
        f"Driver={driver_name};server={server};username=asdf;password=asdf"
    )
    crsr = cnxn.cursor()
    crsr.prepare("SELECT * FROM Pizzas WHERE qqq = 1")
    # pypyodbc doesn't allow us to execute the prepares statements
    # unless we call the SQLExecute function directly
    ret = pypyodbc.SQLExecute(crsr.stmt_h)
    assert ret == pypyodbc.SQL_ERROR
    with pytest.raises(pypyodbc.Error) as exception:
        # Because SQLExecute was updated directly, also call:
        pypyodbc.check_success(crsr, ret)
    assert "Unable to fetch parts" in exception.value.args[1]
    assert "invalid filter column" in exception.value.args[1]


@pytest.mark.parametrize(
    "_",
    [
        None,
        pytest.lazy_fixture("part_resource"),
        pytest.lazy_fixture("part_parameters_resource"),
    ],
)
def test_conditional_select_resource_error(
    httpserver, driver_name, token_resource, categories_resource, _
):
    server = httpserver.url_for("")
    cnxn = pypyodbc.connect(
        f"Driver={driver_name};server={server};username=asdf;password=asdf"
    )
    crsr = cnxn.cursor()
    crsr.prepare("SELECT * FROM Resistors WHERE pk = 1")
    # pypyodbc doesn't allow us to execute the prepares statements
    # unless we call the SQLExecute function directly
    ret = pypyodbc.SQLExecute(crsr.stmt_h)
    assert ret == pypyodbc.SQL_ERROR
    with pytest.raises(pypyodbc.Error) as exception:
        # Because SQLExecute was updated directly, also call:
        pypyodbc.check_success(crsr, ret)
    assert "Unable to fetch parts" in exception.value.args[1]


def test_conditional_select(
    httpserver,
    driver_name,
    token_resource,
    categories_resource,
    part_resource,
    part_parameters_resource,
):
    # TODO: check the category in query string for parts request
    server = httpserver.url_for("")
    cnxn = pypyodbc.connect(
        f"Driver={driver_name};server={server};username=asdf;password=asdf"
    )
    crsr = cnxn.cursor()
    crsr.prepare("SELECT * FROM Resistors WHERE pk = 30")
    # pypyodbc doesn't allow us to execute the prepares statements
    # unless we call the SQLExecute function directly
    ret = pypyodbc.SQLExecute(crsr.stmt_h)
    if ret != pypyodbc.SQL_SUCCESS:
        pypyodbc.check_success(crsr, ret)
    assert (
        ret == pypyodbc.SQL_SUCCESS
    )  # redundant, but what we actually are trying to do
    # Because SQLExecute was updated directly, also call:
    pypyodbc.check_success(crsr, ret)
    crsr._NumOfRows()
    crsr._UpdateDesc()

    results = crsr.fetchall()
    assert len(results) == 1


@pytest.mark.skipif(
    sys.platform == "darwin" and platform.machine() == "x86_64",
    reason="suddenly started failing on amd64+macos+github actions only",
)
@pytest.mark.parametrize(
    "expected",
    [
        pytest.param(
            "'pk' is not a number", marks=pytest.mark.part_mods([("/0/pk", "sixteen")])
        ),
        pytest.param(
            "was unable to convert 'pk' to an int64",
            marks=pytest.mark.part_mods([("/0/pk", 16.1)]),
        ),
    ],
)
def test_invalid_part_pk(
    httpserver,
    driver_name,
    token_resource,
    categories_resource,
    parts_resource,
    expected,
):
    server = httpserver.url_for("")
    cnxn = pypyodbc.connect(
        f"Driver={driver_name};server={server};username=asdf;password=asdf"
    )
    crsr = cnxn.cursor()
    crsr.prepare("SELECT * FROM Resistors")
    # pypyodbc doesn't allow us to execute the prepares statements
    # unless we call the SQLExecute function directly
    ret = pypyodbc.SQLExecute(crsr.stmt_h)
    assert ret == pypyodbc.SQL_ERROR  # redundant, but what we actually are trying to do
    with pytest.raises(pypyodbc.Error) as exception:
        # Because SQLExecute was updated directly, also call:
        pypyodbc.check_success(crsr, ret)
    assert "HY000" == exception.value.args[0]
    assert "Unable to fetch parts" in exception.value.args[1]
    assert expected in exception.value.args[1]


@pytest.mark.skipif(
    sys.platform == "darwin" and platform.machine() == "x86_64",
    reason="suddenly started failing on amd64+macos+github actions only",
)
@pytest.mark.parametrize(
    "expected",
    [
        pytest.param(
            "'pk' is not a number", marks=pytest.mark.part_mods([("/0/pk", "sixteen")])
        ),
        pytest.param(
            "was unable to convert 'pk' to an int64",
            marks=pytest.mark.part_mods([("/0/pk", 16.1)]),
        ),
    ],
)
def test_invalid_parts_pk(
    httpserver,
    driver_name,
    token_resource,
    categories_resource,
    parts_resource,
    part_resource,
    part_parameters_resource,
    expected,
):
    server = httpserver.url_for("")
    cnxn = pypyodbc.connect(
        f"Driver={driver_name};server={server};username=asdf;password=asdf"
    )
    crsr = cnxn.cursor()
    crsr.prepare("SELECT * FROM Resistors WHERE IPN = '123'")
    # pypyodbc doesn't allow us to execute the prepares statements
    # unless we call the SQLExecute function directly
    ret = pypyodbc.SQLExecute(crsr.stmt_h)
    assert ret == pypyodbc.SQL_ERROR
    with pytest.raises(pypyodbc.Error) as exception:
        # Because SQLExecute was updated directly, also call:
        pypyodbc.check_success(crsr, ret)
    assert "HY000" == exception.value.args[0]
    assert "Unable to fetch parts" in exception.value.args[1]
    assert expected in exception.value.args[1]


@pytest.mark.parametrize(
    "table, expected",
    [
        (
            None,
            [
                (None, None, "Capacitors", "TABLE", None),
                (None, None, "Capacitors/Aluminium", "TABLE", None),
                (None, None, "Resistors", "TABLE", None),
                (None, None, "Resistors/NTC", "TABLE", None),
            ],
        ),
        ("Capacitors/Aluminium", [(None, None, "Capacitors/Aluminium", "TABLE", None)]),
        ("Capacitors/Kryptonite", []),
    ],
)
def test_tables(
    httpserver, driver_name, token_resource, categories_resource, table, expected
):
    server = httpserver.url_for("")
    cnxn = pypyodbc.connect(
        f"Driver={driver_name};server={server};username=asdf;password=asdf"
    )
    tables = cnxn.cursor().tables(table=table)

    assert set(tables) == set(expected)
