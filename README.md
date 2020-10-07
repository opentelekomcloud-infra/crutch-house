# Crutch Box
[![Build Status](https://zuul.eco.tsi-dev.otc-service.com/api/tenant/eco/badge?project=opentelekomcloud-infra/crutch-house&pipeline=gate)](https://zuul.eco.tsi-dev.otc-service.com/t/eco/builds?project=opentelekomcloud-infra%2Fcrutch-house&pipeline=gate)
[![codecov](https://codecov.io/gh/opentelekomcloud-infra/crutch-house/branch/master/graph/badge.svg)](https://codecov.io/gh/opentelekomcloud-infra/crutch-house)

Crutch box is a Go library of high-level helper functions for OpenTelekomCloud

It is made of three parts:
1. Port of `github.com/gophercloud/utils` adapted for use with OTC
2. Port of `github.com/docker/machine/libmachine/ssh` but without docker machine itself
3. 100% genuine high-level methods for creating infrastructure in OTC

Why _crutch box_?

Crutch translates to Russian as `костыль (kostýlʹ)` and used as synonym to _workaround_.
This library is nowhere around being beautiful solution and clearly seems to be a workaround,
so it was not a big choice of naming.
