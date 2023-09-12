
This is an Encore application with a single API endpoint to convert Apple Music links to Spotify link and vice versa. I have it set up to work with IOS shortcut - see demo below:

https://github.com/dangxcx/apple2spotify/assets/77860623/3979189e-a388-4b80-b998-c9b397731e82

## Setup

When you have installed Encore, you clone this repro and create a new Encore application 

```bash
encore app create [name]
```


## Running

```bash
# Run the app
encore run
```

## Using the API

To see that your app is running, you can ping the API.

```bash
curl http://localhost:4000/convert -x POST blahlbah
```

## Deployment

Deploy your application to a staging environment in Encore's free development cloud.

```bash
git push encore
```
