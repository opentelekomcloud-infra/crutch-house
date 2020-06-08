/*
Package secgroups enables Querying of SecurityGroups
Security Group service.

Example Querying Security Groups

	listOpts := secgroups.ListOpts{
		Limit: "5",
		VpcID: vpcId,
	}
	sg, err := secgroups.List(vpcClient, listOpts)
	if err != nil {
		panic(err)
	}

	for _, secGrp := range sg {
		fmt.Printf("%+v\n", secGrp)
	}

*/
package secgroups
