# Firebase ThingsDB Module (Go)

Firebase module written using the [Go language](https://golang.org).

Supported:

- Firebase Cloud Messaging


## Installation

Install the module by running the following command in the `@thingsdb` scope:

```javascript
new_module("firebase", "github.com/thingsdb/module-go-firebase");
```

Optionally, you can choose a specific version by adding a `@` followed with the release tag. For example: `@v0.1.0`.

## Configuration

The firebase module requires configuration with the following properties:

Property    | Type             | Description
----------- | ---------------- | -----------
credentials | thing (required) | A service account or refresh token JSON credentials.

Example configuration:

```javascript
// Note: We used 'json_load()' to convert a JSON string into ThingsDB value.
set_module_conf("firebase", {
    credentials: json_load('<paste json here>')
});
```

## Exposed functions

Name                                              | Description
------------------------------------------------- | -----------
[send_message](#send-message)                     | Send a message to one registration token
[send_multicast_message](#send-multicast-message) | Send a message to multiple registration tokens

### Send message

Syntax: `send_message(body, data, title, token)`

#### Arguments

- `body`: _(str)_ Body of the message.
- `data`: _(thing)_ Collection of key-value pairs that will be added to the message as data fields..
- `title`: _(str)_ Title of the message.
- `token`: _(str)_ The registration token of the device to which the message should be sent.

#### Example:

```javascript
body = "Message body.";
data = {
    key: "value"
};
title = "Message title";
token = "<paste token here>";

// Send a notification
firebase.send_message(
    body,
    data,
    title,
    token
).then(|res| {
    res;  // response as "mpdata"
}).else(|err| {
    err;
});
```

### Send multicast message

Syntax: `send_multicast_message(body, data, title, tokens)`

#### Arguments

- `body`: _(str)_ Body of the message.
- `data`: _(thing)_ Collection of key-value pairs that will be added to the message as data fields..
- `title`: _(str)_ Title of the message.
- `tokens`: _(str)_ The registration tokens for the devices to which the message should be distributed.

#### Example:

```javascript
body = "Message body.";
data = {
    key: "value"
};
title = "Message title";
tokens = [
    '<paste token 1 here>',
    '<paste token 2 here>',
];

// Send a notification to multiple registration tokens
firebase.send_multicast_message(
    body,
    data,
    title,
    tokens
).then(|res| {
    res;  // response as "mpdata"
}).else(|err| {
    err;
});
```
