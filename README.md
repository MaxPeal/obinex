Automatically execute OS binaries on the real hardware.

# Installation

    go install gitlab.cs.fau.de/luksen/obinex/...

`obinex-server/weblog.html` needs to be copied to the directory where
`obinex-server` is run.

# Usage

Run `obinex-server` on a buddy PC and (for now) run `obinex` on the same server.

To queue up a binary put it in
`/proj/i4invasic/obinex/<boxname>/in/<somedir>/`. Once the hardware box is free
the binary will run and, along with its output, be placed in
`/proj/i4invasic/obinex/<boxname>/out/<somedir>_YYYY_MM_DD_hh_mm/`.

You can look at the current output of any hardware box at
`http://<buddy>.informatik.uni-erlangen.de:12334/`.

# Internals

## Todo

- test `obinex` on NFS-Server
- other output methods
- clean up output

## Architecture

       +--------------------------+         +-----------------------+        +---------+
       |        Gateway PC        |         |    Buddy PCs          | serial |   HW    |
       +--------------------------+         +-----------------------+ and    +---------+
       |                          |         |                       | http   |         |
       |  +---------------+       |         |  +-----------------+  |        |  +---+  |
       |  |               |       |         |  |  obinex server  <-------------->   |  |
       |  |  +------+  +--+---+   |         |  +-----------------+  |        |  +---+  |
    <-----+--+mail  |  |some  |   |         |                       |        |         |
       |     |      |  |notifier  |         |  +-----------------+  |        |  +---+  |
       |     +---^--+  +--^---+   |         |  |  obinex server  <-------------->   |  |
       |         |        |       |         |  +-----------------+  |        |  +---+  |
       |     +---+--------+---+   |   RPC   |                       |        |         |
       |     |     obinex     <------------->  +-----------------+  |        |  +---+  |
       |     +---^---------^--+   |         |  |  obinex server  <-------------->   |  |
       |         |         |      |         |  +-----------------+  |        |  +---+  |
       |     +---+--+   +--v--+   |         |                       |        |         |
       |     |Magic |   |     |   |         +-----------+-----------+        +---------+
       |     |dirs  |   +-----+   |                     |
    --------->      |   |     |   |                     |Websocket
       |     |      |   +-----+   |                     |
       |     |      |   |     |   |    +----------------v-----+
       |     +------+   +-----+   |    |  Browser/JavaScript  |
       |                          |    +---+------------------+
       +--------------------------+        |
                                           |
    <--------------------------------------+
