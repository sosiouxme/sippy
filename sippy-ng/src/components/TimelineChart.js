import * as d3scale from 'd3-scale'
import PropTypes from 'prop-types'
import React, { useEffect, useRef } from 'react'
import TimelinesChart from 'timelines-chart'

// TimelineChart is a React component to wrap the plain TimelinesChart we used
// in origin previously.
export default function TimelineChart({
  eventIntervals,
  data,
  segmentClickedFunc,
  segmentTooltipContentFunc,
}) {
  const ref = useRef(null)
  const ordinalScale = d3scale
    .scaleOrdinal()
    .domain([
      'InterestingEvent',
      'PathologicalKnown',
      'PathologicalNew', // interesting and pathological events
      'AlertInfo',
      'AlertPending',
      'AlertWarning',
      'AlertCritical', // alerts
      'OperatorUnavailable',
      'OperatorDegraded',
      'OperatorProgressing', // operators
      'Update',
      'Drain',
      'Reboot',
      'OperatingSystemUpdate',
      'NodeNotReady', // nodes
      'Passed',
      'Skipped',
      'Flaked',
      'Failed', // tests
      'PodCreated',
      'PodScheduled',
      'PodTerminating',
      'ContainerWait',
      'ContainerStart',
      'ContainerNotReady',
      'ContainerReady',
      'ContainerReadinessFailed',
      'ContainerReadinessErrored',
      'StartupProbeFailed', // pods
      'CIClusterDisruption',
      'Disruption', // disruption
      'Degraded',
      'Upgradeable',
      'False',
      'Unknown',
      'PodLogInfo',
      'PodLogWarning',
      'PodLogError',
      'EtcdOther',
      'EtcdLeaderFound',
      'EtcdLeaderLost',
      'EtcdLeaderElected',
      'EtcdLeaderMissing',
      'GracefulShutdownInterval',
      'CloudMetric',
    ])
    .range([
      '#6E6E6E',
      '#0000ff',
      '#d0312d', // pathological and interesting events
      '#fada5e',
      '#fada5e',
      '#ffa500',
      '#d0312d', // alerts
      '#d0312d',
      '#ffa500',
      '#fada5e', // operators
      '#1e7bd9',
      '#4294e6',
      '#6aaef2',
      '#96cbff',
      '#fada5e', // nodes
      '#3cb043',
      '#ceba76',
      '#ffa500',
      '#d0312d', // tests
      '#96cbff',
      '#1e7bd9',
      '#ffa500',
      '#ca8dfd',
      '#9300ff',
      '#fada5e',
      '#3cb043',
      '#d0312d',
      '#d0312d',
      '#c90076', // pods
      '#96cbff',
      '#d0312d', // disruption
      '#b65049',
      '#32b8b6',
      '#ffffff',
      '#bbbbbb',
      '#96cbff',
      '#fada5e',
      '#d0312d',
      '#d3d3de',
      '#03fc62',
      '#fc0303',
      '#fada5e',
      '#8c5efa',
      '#6E6E6E', // GracefulShutdownInterval
      '#6E6E6E', // CloudMetric
    ])

  useEffect(() => {
    let chart = null
    const node = ref.current
    if (node) {
      //const el = document.querySelector('#chart')
      chart = TimelinesChart()(node)
        .data(data)
        .useUtc(true)
        .zQualitative(true)
        .enableAnimations(false)
        .enableOverview(false)
        .leftMargin(150)
        .rightMargin(750)
        .maxLineHeight(30)
        .maxHeight(20000)
        .onSegmentClick(segmentClickedFunc)
        .segmentTooltipContent(segmentTooltipContentFunc)
        .zColorScale(ordinalScale) // seems to enable the use of our own colors
      if (eventIntervals.length > 0) {
        chart.zoomX([
          new Date(eventIntervals[0].from),
          new Date(eventIntervals[eventIntervals.length - 1].to),
        ])
      }
    }
    return () => {
      if (node) {
        while (node.firstChild) {
          node.removeChild(node.firstChild)
        }
      }
    }
  }, [data])
  return React.createElement('div', { ref: ref })
}

TimelineChart.propTypes = {
  data: PropTypes.array,
  eventIntervals: PropTypes.array,
  segmentClickedFunc: PropTypes.func,
  segmentTooltipContentFunc: PropTypes.func,
}
