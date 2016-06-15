Automatically execute OS binaries on the real hardware.

       +--------------------------+          +-----------------------+        +---------+
       |        Gateway PC        |   RPC    |    Buddy PCs          | serial |   HW    |
       +--------------------------+host      +-----------------------+ and    +---------+
       |                          |   clients|                       | http   |         |
       |  +---------------+       |          |  +-----------------+  |        |  +---+  |
       |  |               |       |       +----->  obinex-server  <-------------->   |  |
       |  |  +------+  +--+---+   |       |  |  +-----------------+  |        |  +---+  |
    <-----+--+mail  |  |some  |   |       |  |                       |        |         |
       |     |      |  |notifier  |       |  |  +-----------------+  |        |  +---+  |
       |     +---^--+  +--^---+   |       +----->  obinex-server  <-------------->   |  |
       |         |        |       |       |  |  +-----------------+  |        |  +---+  |
       |     +---+--------+---+   |       |  |                       |        |         |
       |     |     obinex     <-----------+  |  +-----------------+  |        |  +---+  |
       |     +---^---------^--+   |       +----->  obinex-server  <-------------->   |  |
       |         |         |      |          |  +-----------------+  |        |  +---+  |
       |     +---+--+   +--v--+   |          |                       |        |         |
       |     |Magic |   |     |   |          +-----------------------+        +---------+
       |     |dirs  |   +-----+   |
    --------->      |   |     |   |
       |     |      |   +-----+   |
       |     |      |   |     |   |
       |     +------+   +-----+   |
       |                          |
       +--------------------------+
