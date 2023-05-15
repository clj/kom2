# Prototype

ODBC to InvenTree prototype driver for KiCad.

See also: https://github.com/clj/kom for a previous prototype that uses SQLite virtual tables

## Example config

inventree.kicad_dbl
```json
{
    "meta": {
        "version": 0
    },
    "name": "Inventree Library",
    "description": "Components pulled from Inventree",
    "source": {
        "type": "odbc",
        "dsn": "kom2",
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

~/.odbc.ini

```ini
[kom2]
Driver=kom2.dylib
Description=Kom2 test
Server=https://...
UID=my_username
PASSWORD=API_KEY
```
