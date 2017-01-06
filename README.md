# go-odbye
Goodbye unfollowers!

A simple tool to check who [un]follow you on twitter.

## Dependencies
To use this tool you need to resolve some dependencies like go-twitter, oauth1, color and go-sqlite3, to do this enter on the directory `cd $GOPATH/src/go-odbye` and type `go get`.

If you run into trouble, this is commonly an error pertaining to `int()` or `int64()`, you should update your go dependancies with `go get -u`.

## Building
In the folder `$GOPATH/src/go-odbye` you need to run `go build` to compile the binary `go-odbye`.

## Usage
`./go-odbye -nick <yournick>`

## Example
`./go-odbye -nick dlion92`

![screenshot](./screenshot.png)

## Option
`-nick <nick> Which user you want to track`   
`-url <true|false> If you want to see complete url`

## Config
To use this tool you need to add your consumer and token key/secret in a json config file on your `$HOME` dir called `.goodbye.json`, here the structure:   
```js
{
  "consumerKey": "<CONSUMER KEY>",
  "consumerSecret": "<CONSUMER SECRET>",
  "accessToken": "<ACCESS TOKEN>",
  "accessSecret": "<ACCESS SECRET>"
}
```

You can also optionally set your default nick in the config file
```js
{
  "nick": "<yournick>",
  // …
}
```

## Consumer/Token
You can obtain your consumer/token creating a new twitter application, see https://apps.twitter.com/ to more info.

## Author
Domenico Luciani aka DLion
https://domenicoluciani.com

## License
MIT
