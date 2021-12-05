# insync

Get telegram notifications if your geth node looses sync 

# environment variables
- GETH_URL = the url of your node
- BOT_TOKEN = your telegram bot token
- CHECK_INTERVAL = the interval to check (e.g. 5s)
- REPORT_INTERVAL = the interval to report (if the node was never in sync during that timeframe)
- ALERT_GROUP = the group or user to send alerts to