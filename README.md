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
`/proj/i4invasic/obinex/<boxname>/out/<somedir>/<binary-name>_YYYY_MM_DD_hh_mm/`.

You can look at the current output of any hardware box at
`http://<buddy>.informatik.uni-erlangen.de:12334/`.

# Internals

## Architecture

       +--------------------------+         +---------------------------+        +---------+
       |        Gateway PC        |         |        Buddy PCs          | serial |   HW    |
       +--------------------------+         +---------------------------+ and    +---------+
       |                          |         |                           | http   |         |
       |  +---------------+       |         | +-------+---------------+ |        |  +---+  |
       |  |               |       |         | | queue | obinex server <------------->   |  |
       |  |  +------+  +--+---+   |         | +-------+---------------+ |        |  +---+  |
    <-----+--+mail  |  |some  |   |         |                           |        |         |
       |     |      |  |notifier  |         | +-------+---------------+ |        |  +---+  |
       |     +---^--+  +--^---+   |         | | queue | obinex server <------------->   |  |
       |         |        |       |         | +-------+---------------+ |        |  +---+  |
       |     +---+--------+---+   |   RPC   |                           |        |         |
       |     |     obinex     <-------------> +-------+---------------+ |        |  +---+  |
       |     +---^------------+   |         | | queue | obinex server <------------->   |  |
       |         |                |         | +-------+---------------+ |        |  +---+  |
       |     +---+--+             |         |                           |        |         |
       |     |Magic |             |         +-----------+---------------+        +---------+
       |     |dirs  |             |                     |
    --------->      |             |                     |Websocket
       |     |      |             |                     |
       |     |      |             |    +----------------v-----+
       |     +------+             |    |  Browser/JavaScript  |
       |                          |    +---+------------------+
       +--------------------------+        |
                                           |
    <--------------------------------------+
