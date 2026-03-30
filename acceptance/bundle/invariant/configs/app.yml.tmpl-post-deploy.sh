# Run the app after the deploy otherwise migrate will show the drift on the source code path.
# This happens because source code path is set remotely only after the deploy.
$CLI bundle run foo
