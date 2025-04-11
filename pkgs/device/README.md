# Device SDK

## Overview

## Device Storage

BoltDB to store desired properties and messages.

Properties are both a cache and a persistent storage for device. Properties name and

### Properties

Bucket name is 'properties' for config. On bootup, load the device's configuration from the storage. On new config arrival,
update the cache and bucket with the new config

### Messages

Bucket name is 'msgs' for messages.

Flow:
  Messages are all stored in the bucket with a timelimit and storage limit. Msgs are not stored in cache.
  Reported properties stored? NO
  What do we store? The data or the full nats msg. FULL

Option1: All msgs are stored.
Option2: Stored only if could not be sent
Option3: No storage

Two buckets: one for sent, one for pending
When pending is sent, move it to sent if Option 1

Key/Value format
key = Time RFC3339
value = nats msg

On bootup, get the buckets for writing, check if time or disk limit is reached and delete old msgs.

Task to clear old messages every 1h. Can be a setting

On connect, read all values from pending bucket and sent them to server and delete. If save option is enabled, save the sent msg to sent bucket.

If connected, sent msg to server and sent bucket if Option 1.
If disconnected, sent msg to pending bucket.
Change writing handler for most effiency.
On reconnect, redo properties handler?


### Limits

Rentention and DiskSpace

Flow:
 - On online connect, delete persistent on dates
 - On offline connect, delete persistent and pending on dates
 - After, batch of 100 msgs, delete at a time and check diskpace, persistent first then pending
 - Task every hour that does this
