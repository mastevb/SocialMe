# SocialMe: A Google Cloud Based Social Network Web Application
## About me
Hi! My name is Steve, and I'm a 4th year B.S./M.S. student at [UWCSE](https://www.cs.washington.edu)! Feel free to connect me on [LinkedIn](https://www.linkedin.com/in/steve-ma/) or send me an [email](mailto:%20bochenma@cs.washington.edu), also check out my [website](https://mastevb.github.io/steve_ma_uwcse.io/)!

## About SocialMe
![Photos](https://github.com/mastevb/SocialMe/blob/master/socialme-web/src/assets/images/Screen%20Shot%202020-08-02%20at%208.44.03%20PM.png)
SocialMe is a Google Cloud based, React based and Go based social network web application for connecting people around by shared photos and videos. This idea was inspired by the "nearme" feature in TikTok.

## Why Go?
Go is a statically typed, compiled programming language designed at Google. Go is syntatically similar to C, but with memory safety, garbage collection, and structural typing.
So why did I choose to use Go?
* Go is believed to be the server language for next-generation, the language is efficient for execution and the learning bar is lower than traditional languages such as C. Go is compiled to machine code and is executed directly, much like C and unlike Java.
* Go is also an awesome language for concurrency. Although I don't have much experience in concurrent computing, there's a few things that Go did right
    * Goroutines are cheap and lightweighted comparing to threads. Goroutines are only a few kb in stack size, and the stack can grow and shrink according to needs of the application, comparing to the case of threads where the stack size has to be specified and is fixed.
    * Goroutines are multiplexed to fewer number of OS threads. One thread in the OS can corresponds to thousands of Goroutines.
    * Goroutines communicate through channels, and channels are built into the language.Channels remove the need for more explicit locking and thus is easier to write correctly, tune for perforemance and ebug.

## Google Vision API

## Elasticsearch engine

## Google Cloud Storage

## Google Map API
![Photos](https://github.com/mastevb/SocialMe/blob/master/socialme-web/src/assets/images/Screen%20Shot%202020-08-02%20at%208.45.03%20PM.png)

## Token Based Authentication

* React Router v4

* JWT

##  Google Kubernetes Engine

## AWS Amplify
