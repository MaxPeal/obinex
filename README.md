Automatically execute OS binaries on the real hardware.

       +--------------------------+          +-------------------+        +---------+
       |        Gateway PC        |   RPC    |    Buddy PCs      | serial |   HW    |
       +--------------------------+host      +-------------------+ and    +---------+
       |                          |   clients|                   | http   |         |
       |  +---------------+       |          |  +-------------+  |        |  +---+  |
       |  |               |       |       +----->  server.go  <-------------->   |  |
       |  |  +------+  +--+---+   |       |  |  +-------------+  |        |  +---+  |
    <-----+--+mail  |  |some  |   |       |  |                   |        |         |
       |     |      |  |notifier  |       |  |  +-------------+  |        |  +---+  |
       |     +---^--+  +--^---+   |       +----->  server.go  <-------------->   |  |
       |         |        |       |       |  |  +-------------+  |        |  +---+  |
       |     +---+--------+---+   |       |  |                   |        |         |
       |     |   service.go   <-----------+  |  +-------------+  |        |  +---+  |
       |     +---^---------^--+   |       +----->  server.go  <-------------->   |  |
       |         |         |      |          |  +-------------+  |        |  +---+  |
       |     +---+--+   +--v--+   |          |                   |        |         |
       |     |Magic |   |     |   |          +-------------------+        +---------+
       |     |dirs  |   +-----+   |
    --------->      |   |     |   |
       |     |      |   +-----+   |
       |     |      |   |     |   |
       |     +------+   +-----+   |
       |                          |
       +--------------------------+