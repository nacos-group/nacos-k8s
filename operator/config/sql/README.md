# Nacos MySQL schema

`nacos-mysql.sql` is the unmodified schema for Nacos Server **2.4.3**:

- Source: https://github.com/alibaba/nacos/blob/2.4.3/distribution/conf/mysql-schema.sql
- SHA-256: `87aaae43774601ff5223a385de9f0d64840b5b7ec59fb26217e955a276afdb21`

The Operator image copies this file from the build context instead of downloading
SQL from the moving `develop` branch. Keep the original license header and update
this source and checksum if the bundled schema is intentionally changed.

This file describes the schema bundled with the image; it is not a database
migration or a request to execute SQL against an existing database.
