package terraform

// func TestSomething(t *testing.T) {
// 	ctx := context.Background()
// 	tf, err := newTerraform(ctx, "testdata/simplest", map[string]string{
// 		"DATABRICKS_HOST":  "..",
// 		"DATABRICKS_TOKEN": "..",
// 	})
// 	assert.NoError(t, err)

// 	err = tf.Init(ctx)
// 	assert.NoError(t, err)

// 	planLoc := path.Join(t.TempDir(), "tfplan.zip")
// 	hasChanges, err := tf.Plan(ctx, tfexec.Out(planLoc))
// 	assert.True(t, hasChanges)
// 	assert.NoError(t, err)

// 	plan, err := tf.ShowPlanFile(ctx, planLoc)
// 	assert.NoError(t, err)
// 	assert.NotNil(t, plan)

// 	found := false
// 	for _, r := range plan.Config.RootModule.Resources {
// 		// TODO: add validator to prevent non-Databricks resources in *.tf files, as
// 		// we're not replacing terraform, we're wrapping it up for the average user.
// 		if r.Type != "databricks_job" {
// 			continue
// 		}
// 		// TODO: validate that libraries on jobs defined in *.tf and libraries
// 		// in `install_requires` defined in setup.py are the same. Exist with
// 		// the explanatory error otherwise.
// 		found = true
// 		// resource "databricks_job" "this"
// 		assert.Equal(t, "this", r.Name)
// 		// this is just a PoC to show how to retrieve DBR version from definitions.
// 		// production code should perform rigorous nil checks...
// 		nc := r.Expressions["new_cluster"]
// 		firstBlock := nc.NestedBlocks[0]
// 		ver := firstBlock["spark_version"].ConstantValue.(string)
// 		assert.Equal(t, "10.0.1", ver)
// 	}
// 	assert.True(t, found)
// }
