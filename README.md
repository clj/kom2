# Prototype

ODBC to InvenTree prototype driver for KiCad.

See also: https://github.com/clj/kom for a previous prototype that uses SQLite virtual tables

## Installing

### macOS

Install [unixODBC](https://www.unixodbc.org)

```shell
$ brew install unixodbc
```


Download the latest kicad-odbc-middleware2 release from [Releases](https://github.com/clj/kom2/releases):

* For Intel based Macs:
    * kicad-odbc-middleware-**macos**-**amd64**-VERSION.zip
* For Apple Silicon (ARM) Macs:
    * kicad-odbc-middleware-**macos**-**arm64**-VERSION.zip

decompress it and leave the kom2.dylib file somewhere convenient.

### KiCad Configuration

Create a `inventree.kicad_dbl` file with a valid configuration (see the [KiCad documentation on Database Libraries](https://docs.kicad.org/master/en/eeschema/eeschema.html#database-libraries)), e.g.:

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

### Add the library to KiCad:

* *Preferences* -> *Manage Symbol Libraries...*
* Switch to the:
    * *Global Libraries*; or
    * *Project Specific Libraries*
* Add a new library
* Give it an appropriate *Nickname*
* Set the *Library Path* to point to the `inventree.kicad_dbl` that you created earlier
* Set the *Library Format* to *Database*

You can now open the Schematic Editor and add a new component. The configured library should now be available.

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

## License

MIT License Copyright (c) 2023 Christian Lyder Jacobsen

Refer to [LICENSE](./LICENSE) for full text.