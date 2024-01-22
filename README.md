# Rayscan

Working demo for [this guide](https://github.com/patrulek/Writings/blob/main/Discovering%20Raydium%20Pairs.md)

## Build and run

If you want to build it from source, download Go toolchain and put

```console
go build main.go
```

in the console, then run executable.

If you want to run it from source, download Go toolchain and put

```console
go run main.go
```

in the console.

If you just want to run it, there's `main.exe` executable in the repo (only for Windows).

## Configuration

`config.toml` is provided to configure RPC nodes tool will connect to. You can set RPC endpoint, websocket endpoint and observer flag, which is used to enable transcation logs retrieval from given node.

## Sample output

```console
Checking connection list...
Connection rpcpool-hxro is healthy
Connection tatum-us-ms-matter is healthy
Connection helius-masterhoneysuckle is healthy
Connection extrnode-testiiiing is healthy
Connection mainnet is healthy
Connection list checked! 5/5 connections are ok [rpcpool-hxro, tatum-us-ms-matter, helius-masterhoneysuckle, extrnode-testiiiing, mainnet]
[2024-01-22 20:11:09.168] PairCollector: starting...
[2024-01-22 20:11:09.175] TxAnalyzer: starting...
[2024-01-22 20:11:09.175] LogObserver: Subscribe for OpenBook program logs on helius-masterhoneysuckle...
[2024-01-22 20:11:09.368] LogObserver: Subscribe for Raydium Liquidity program logs on helius-masterhoneysuckle...
[2024-01-22 20:11:09.545] LogObserver: Subscribe for OpenBook program logs on extrnode-testiiiing...
[2024-01-22 20:11:09.797] LogObserver: Subscribe for Raydium Liquidity program logs on extrnode-testiiiing...
[2024-01-22 20:11:10.073] LogObserver: Subscribe for OpenBook program logs on mainnet...
[2024-01-22 20:11:10.186] LogObserver: Subscribe for Raydium Liquidity program logs on mainnet...
[2024-01-22 20:11:21.614] PairCollector: new market discovered for (token: 7dDusCM8r5G2aPpF1RT24JyjAHsHAZRsRpwSChUm4tjy, tx time: 2024-01-22 20:11:20.000)
[2024-01-22 20:11:22.592] PairCollector: new market discovered for (token: DM6gk5sTqbYEEwPKNfVRcr1N3rXsBTMJb45RmsaVWxhg, tx time: 2024-01-22 20:11:19.000)
[2024-01-22 20:11:38.657] PairCollector: new market discovered for (token: pQfC7zMTjdTnjxKTDGWj5SudJx7E2ypmcFzgdV7Q6eg, tx time: 2024-01-22 20:11:37.000)
[2024-01-22 20:11:56.327] PairCollector: new pair found (token: pQfC7zMTjdTnjxKTDGWj5SudJx7E2ypmcFzgdV7Q6eg, ammid: 9EeqQMSyySiLz2JYPwRyVFntJNJY5L1PLUFfnVwCs8hx, opentime: 2024-01-22 20:11:53.000)        
[2024-01-22 20:12:07.768] PairCollector: error handling info (*raydium.AmmInfo): amm, no pair for token: GXtRrXjAckggMAgfExa7cbFrgNWgU8X1WS7kJuzqPgJr
[2024-01-22 20:12:09.019] PairCollector: new market discovered for (token: 5fbHoLHWykRMwakz6YA49zj3ZrQ5rHoYtW6JtAzVk4Xx, tx time: 2024-01-22 20:12:07.000)
[2024-01-22 20:12:26.097] PairCollector: new market discovered for (token: EB718W6KTiTwubzSVZ1QZ17roxYc7XHS1jJu6cYEjS2D, tx time: 2024-01-22 20:12:25.000)
[2024-01-22 20:12:38.587] PairCollector: new market discovered for (token: 4fEoFVZWmjtEGb1WigmCaTvXtto4ytrqfDjDcvKawBKA, tx time: 2024-01-22 20:12:37.000)
[2024-01-22 20:12:54.086] PairCollector: error handling info (*raydium.AmmInfo): amm, no pair for token: Cz6EJqn1przUPAmsohFZULNNWoJvv2hx5fGKNSuNozJc
[2024-01-22 20:13:03.148] PairCollector: new pair found (token: EB718W6KTiTwubzSVZ1QZ17roxYc7XHS1jJu6cYEjS2D, ammid: 9GA2oagDDPmaehh74Lk38L2esXTNwJo831QH6TCRnofM, opentime: 2024-01-22 20:12:56.000)       
[2024-01-22 20:13:03.265] PairCollector: new pair found (token: 5fbHoLHWykRMwakz6YA49zj3ZrQ5rHoYtW6JtAzVk4Xx, ammid: 5okPrzRtL5f76CCwfQXEuY3f9yRJbd6EjCtEw4VwZ5ig, opentime: 2024-01-22 20:12:54.000)       
[2024-01-22 20:13:30.886] PairCollector: new market discovered for (token: CVrKVgrLhqBzbbjNF8uSkLEDk6VNt6D3o8U1riSb2wzV, tx time: 2024-01-22 20:13:29.000)
[2024-01-22 20:13:31.137] PairCollector: new market discovered for (token: AvRsSvbBoz6UoHura22PMDwsH3iwxgaMvodnSPepPW4L, tx time: 2024-01-22 20:13:29.000)
[2024-01-22 20:13:39.381] PairCollector: error handling info (*raydium.AmmInfo): amm, no pair for token: FLvmsxZMHumixsG712xXy7xfBU8sYz7KW18httjqHGZH
[2024-01-22 20:13:51.893] PairCollector: new market discovered for (token: 2zxozeF9xZUm8HezddeXVRfgiigfHcHud5ENdKEJBFnf, tx time: 2024-01-22 20:13:47.000)
[2024-01-22 20:14:04.888] PairCollector: error handling info (*raydium.AmmInfo): amm, no pair for token: 4Zoks3hUYaCxcKaXbzjgAqb6UGQxrZUFpBQoA9zKnNV5
[2024-01-22 20:14:17.469] PairCollector: new market discovered for (token: 9tuGBBc72oCPvCAMdyD9f4BzPfSPK2ypu81QfLYXCivW, tx time: 2024-01-22 20:14:16.000)
[2024-01-22 20:15:02.826] PairCollector: new market discovered for (token: AP16XZGJGsd7AxSJx65qdvjapAW5paTJxaEhoik5RbcW, tx time: 2024-01-22 20:15:01.000)
[2024-01-22 20:15:23.341] PairCollector: new market discovered for (token: 5wq2x3qx3xUR3UxMkZHav37379wJoKZ64Sw5MY6yUbK5, tx time: 2024-01-22 20:15:21.000)
[2024-01-22 20:15:31.614] PairCollector: new pair found (token: 2zxozeF9xZUm8HezddeXVRfgiigfHcHud5ENdKEJBFnf, ammid: A4DLw6TyerwpputBFm2EwhaZoCEicLuMVMgR5AFTFswx, opentime: 2024-01-22 20:15:26.000)       
[2024-01-22 20:15:54.939] PairCollector: new pair found (token: AP16XZGJGsd7AxSJx65qdvjapAW5paTJxaEhoik5RbcW, ammid: 49w4WFZwBnUiCZ7HPrfX9QxSJKWkw6K5TjmudpDxhbmi, opentime: 2024-01-22 20:16:00.000)       
[2024-01-22 20:15:57.698] PairCollector: new market discovered for (token: 5U9a5TG8WHzAt82pSGvh5CqaxK3p6bYNEdtiydKYbdm6, tx time: 2024-01-22 20:15:55.000)
[2024-01-22 20:17:45.758] PairCollector: new market discovered for (token: 8u4wNFdUnigSc8N714QUg24LBmMqnV7xUZDwNrwLX8Eq, tx time: 2024-01-22 20:17:44.000)
[2024-01-22 20:17:48.770] PairCollector: new pair found (token: 5wq2x3qx3xUR3UxMkZHav37379wJoKZ64Sw5MY6yUbK5, ammid: 6rjSjdXs5Fx1fbauxFyesrvaUMqPGN2UPcawyhJt6qrN, opentime: 2024-01-22 20:17:29.000)
[2024-01-22 20:18:28.876] PairCollector: new pair found (token: 8u4wNFdUnigSc8N714QUg24LBmMqnV7xUZDwNrwLX8Eq, ammid: FpnPytn67AiBLmUQTcoca4VUESg6MPyGeRBvw3oGCDFT, opentime: 2024-01-22 20:20:00.000)
[2024-01-22 20:19:32.190] PairCollector: new pair found (token: CVrKVgrLhqBzbbjNF8uSkLEDk6VNt6D3o8U1riSb2wzV, ammid: 7PG5imrppD8gWpUETrhLk4Jy3y1m7Ni6NzMo6oFNPfpV, opentime: 2024-01-22 20:19:16.000)
[2024-01-22 20:20:04.706] PairCollector: new market discovered for (token: DzNDpBDTyfnUmjt8MXrN2WUtSA5Pzqxhh5schQ2oK9DM, tx time: 2024-01-22 20:20:03.000)
[2024-01-22 20:20:05.618] PairCollector: new market discovered for (token: DEdXZL9U76mnJZYqAg9ftNnU6ZX2DjQqcYKmqdKa5EQr, tx time: 2024-01-22 20:20:03.000)
[2024-01-22 20:20:53.910] PairCollector: new pair found (token: 4fEoFVZWmjtEGb1WigmCaTvXtto4ytrqfDjDcvKawBKA, ammid: 4vMajVMRCuqC3Whv3S2t28EYGiGDXuR9DGzKScNSbyTH, opentime: 2024-01-22 20:20:42.000)
[2024-01-22 20:20:57.715] PairCollector: new market discovered for (token: 4ZpDNuiLHnC5TinG8WQWFAcRYiW3RCxnshgRfHipppZP, tx time: 2024-01-22 20:20:56.000)
[2024-01-22 20:22:08.725] PairCollector: new market discovered for (token: ER8rbJgTuWsn19p6z1k1N4SazZdhDbpwHDm79bFgzqk, tx time: 2024-01-22 20:22:07.000)
Interrupted; stopping...
```