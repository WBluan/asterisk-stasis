# Asterisk ARI Call Handling Project

This project demonstrates how to use the Asterisk ARI (Asterisk REST Interface) to handle incoming calls, create bridges, and originate outbound calls using the PJSIP channel technology.
Overview

This application listens for incoming calls, answers them, and checks the dialed number. If the dialed number matches a predefined value (in this case, "100"), it creates a bridge and originates calls to two endpoints via PJSIP. All interactions with Asterisk are done through ARI.
Features

* Listens for incoming calls via ARI.
* Answers incoming calls and retrieves dialed number.
* Creates a bridge for call mixing.
* Originates outbound calls to specific PJSIP endpoints.
* Adds channels to a bridge for conferencing.
* Logs important events like call start, end, and errors.

## Requirements

* Asterisk (version with ARI enabled)
* PJSIP channel technology configured in Asterisk
* Go (Golang) installed
* MariaDB for Asterisk backend (optional)
* ARI Client Libraries: This project uses the ari/v6 client library for Go.