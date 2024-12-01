StarDB is an in-memory database. The following are the goals of this repository:

1) Implement a custom communication protocol
2) Allow multiple TCP clients to concurrently interact with the DB instance
3) Understand and implement I/O multiplexing to accommodate a single-threaded database
4) Develop GO experience.
5) Develop hands-on experience w.r.t building a performant service, from scratch



Learnings & Challenges log:

1) Getting a grasp of kqueue: Kqueue is a kernel event notification mechanism provided by various operating systems, including FreeBSD, OpenBSD, NetBSD, and macOS. It allows an application to monitor multiple file descriptors for various events.

2) Handling multiple time-spaced messages from the same connection via attaching a kqueue event to each connection's file descriptor.