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
    * kicad-odbc-middleware2-**macos**-**amd64**-VERSION.zip
* For Apple Silicon (ARM) Macs:
    * kicad-odbc-middleware2-**macos**-**arm64**-VERSION.zip

decompress it and leave the kom2.dylib file somewhere convenient.

### Linux

Contributions welcome.

### Windows

Download the latest kicad-odbc-middleware2 Windows installer from [Releases](https://github.com/clj/kom2/releases) and run it.

* For Intel/Amd based Windows PCs:
  * kicad-odbc-middleware2-**windows**-**amd64**-VERSION.exe

### KiCad Configuration

Create a `inventree.kicad_dbl` file with a valid configuration (see the [KiCad documentation on Database Libraries](https://docs.kicad.org/master/en/eeschema/eeschema.html#database-libraries)), e.g.:

```json
{
    "meta": {
        "version": 0
    },
    "name": "InvenTree Library",
    "description": "Components pulled from InvenTree",
    "source": {
        "type": "odbc",
        "connection_string": "Driver=SEE_BELOW;username=reader;password=readonly;server=https://demo.inventree.org",
        "timeout_seconds": 2
    },
    "libraries": [
        {
            "name": "Resistors",
            "table": "Electronics/Passives/Resistors",
            "key": "pk",
            "symbols": "parameter.Symbol",
            "footprints": "parameter.Footprint",
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

The InvenTree Demo server does not seem to have IPNs for everything though, so the key should probably be `pk` instead if that is the case (i.e. if IPN isn't unique).

#### Connection String

The `Driver` argument of the connection string is used to find the kom2 driver and the exact value used is platform specific.
##### Windows

Use `Driver=kom2` when the driver was installed using the downloaded Windows installer.

##### Linux

Use `Driver=/path/to/kom2.so`, using the correct path where the driver was downloaded and extracted.

##### macOS

Use `Driver=/path/to/kom2.dylib`, using the correct path where the driver was downloaded and extracted.

##### Other Connection String Options

* `username`
    * InvenTree username
* `password`
    * InvenTree users's password (not required if the `apitoken` is used instead)
* `server`
    * The InvenTree server to connect to
* `apitoken`
    * The optional API token (not required when `username` and `password` are used)

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