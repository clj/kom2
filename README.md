# Prototype

ODBC to InvenTree prototype driver for KiCad.

See also: https://github.com/clj/kom for a previous prototype that uses SQLite virtual tables

## Example config

inventree.kicad_dbl:
```json
{
    "meta": {
        "version": 0
    },
    "name": "Inventree Library",
    "description": "Components pulled from Inventree",
    "source": {
        "type": "odbc",
        "connection_string": "Driver=/.../kom2.dylib;username=reader;password=readonly;server=https://demo.inventree.org",
        "timeout_seconds": 2
    },
    "libraries": [
        {
            "name": "Resistors",
            "table": "Resistors",
            "key": "IPN",
            "symbols": "metadata.kicad.symbols",
            "footprints": "metadata.kicad.footprints",
            "fields": [
                {
                    "column": "IPN",
                    "name": "IPN",
                    "visible_on_add": false,
                    "visible_in_chooser": true,
                    "show_name": true,
                    "inherit_properties": true
                },
                {
                    "column": "parameter.Value",
                    "name": "Value",
                    "visible_on_add": true,
                    "visible_in_chooser": true,
                    "show_name": false
                },
                {
                    "column": "parameter.Package",
                    "name": "Package",
                    "visible_on_add": true,
                    "visible_in_chooser": true,
                    "show_name": false
                }
            ],
            "properties": {
                "description": "description",
                "keywords": "keywords"
            }
        }
    ]
}
```

The InvenTree Demo servier does not seem to have IPNs for everything though, so the key should probably be `pk` instead if that is the case (i.e. if IPN isn't unique).

## Interactive Use

You can query InvenTree using `isql` by using a connection string:

```
isql -v -k "Driver=/.../kom2.dylib;username=reader;password=readonly;server=https://demo.inventree.org"
```

    and run things like:

```
select * from Electronics/Passives/Resistors
```

or

```
select * from Electronics/Passives/Resistors where pk = 43;
```

or

```
select * from Electronics/Passives/Resistors where IPN = ???;
```

if there were IPNs in the DB.