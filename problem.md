# The problem

Write a file sending service!

Suppose you have two laptops (A and B) and a server. The laptops are in different 
houses and each are behind a firewall, so laptop A can't talk to laptop B and
vice versa. The server is in a datacenter somewhere, and both laptops can talk
to it.

The user of laptop A wants to send a file to the user of laptop B. The users are 
talking on the phone, so they can exchange information, but not the file itself.

You'll need to write three programs:

 1. **The sender** - this program will run on laptop A.
 2. **The receiver** - this program will run on laptop B.
 3. **The relay** - this program will run on the server, which both laptops can reach.
 
The relay will always be running. Our automated tests will start the relay with the
TCP port the server will listen on.

```
./relay :<port>
```

To send a file, the user of laptop A will run

```
./send <relay-host>:<relay-port> <file-to-send>
```

The sending program will nearly immediately output a secret code that 
the user of laptop A will tell the user of laptop B over the phone. 
The sending program should *only* output the secret code, or our 
automated tests won't work. The sending program will then wait to
send the file.

To receive the file, the user of laptop B will run

```
./receive <relay-host>:<relay-port> <secret-code> <output-directory>
```

As soon as the receive program is run, the sender will start sending
and the receiver will start receiving. As soon as the file is 
completely sent, both programs will then exit.

## Some notes

 * Your relay program should not use much memory. It should not use more than 4MB of memory or storage per transfer, regardless of the size of the file being transfered.
 * The send and receive programs cannot initiate communication with anyone other than the relay program.
 * Your relay program should support multiple people sending files at the same time.
 * You should submit some documentation on how to use and the overall architecture.
 * While this is just an interview problem, you should treat this solution the same way you would treat code that you anticipate deploying and managing in production. Please define in your documentation what important aspects of your submitted code make it production ready.
 * This probably goes without saying for an interview problem but your goal is to impress us with your knowledge and software engineering prowess. Other engineers on the team will be evaluating your submission and your code based on whether the code you generate is the type of code they want to work on and live with.

Use whatever language and tools you feel most comfortable with, though we ask that you try to limit yourself to standard libraries where possible. It's worth pointing out that the Go programming language is an exceptionally good choice for this kind of problem, but please use what you think will help you make the strongest submission.

## Example session

Terminal 1:
```
$ ./relay :9021
```

Terminal 2:
```
$ ./send localhost:9021 corgis.mp4
this-is-a-secret-code
```

Terminal 3:
```
$ ./receive localhost:9021 this-is-a-secret-code out/
$ ls out/
corgis.mp4
$
```

Thanks, and good luck!
