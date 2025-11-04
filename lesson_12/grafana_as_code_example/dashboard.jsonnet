local grafana = import 'grafonnet/grafana.libsonnet';
grafana.dashboard.new(
    title='My dashboard',
    # allow the user to make changes in Grafana
    editable=true,
    # avoid issues associated with importing multiple versions in Grafana
    schemaVersion=21,
).addPanel(
    grafana.graphPanel.new(
        title='My first graph',
        # demonstration data
        datasource='-- Grafana --'
    ),
    # panel position and size
    gridPos = { h: 8, w: 8, x: 0, y: 0 }
)
