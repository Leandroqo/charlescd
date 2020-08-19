/*
 * Copyright 2020 ZUP IT SERVICOS EM TECNOLOGIA E INOVACAO SA
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import React, { useEffect, useState } from 'react';
import Text from 'core/components/Text';
import { useForm } from 'react-hook-form';
import { normalizeCircleParams } from '../helpers';
import { useDeployMetric } from './hooks';
import deployOptions from './deploy.options';
import { periodFilterItems } from './constants';
import Styled from './styled';
import CircleFilter from './CircleFilter';
import ChartMenu from './ChartMenu';
import { getDeploySeries } from './helpers';
import ReleasesHistoryComponent from './Release';
import { ReleaseHistoryRequest } from './interfaces';

const Deploys = () => {
  const { searchDeployMetrics, response, loading } = useDeployMetric();
  const { control, handleSubmit, getValues, setValue } = useForm();
  const deploySeries = getDeploySeries(response);

  useEffect(() => {
    searchDeployMetrics({ period: periodFilterItems[0].value });
  }, [searchDeployMetrics]);

  const [filter, setFilter] = useState<ReleaseHistoryRequest>({
    period: periodFilterItems[0].value,
    circles: []
  });

  const onSubmit = () => {
    const { circles, period } = getValues();
    const circleIds = normalizeCircleParams(circles);
    setFilter({ period, circles: circleIds });
    searchDeployMetrics({ period, circles: circleIds });
  };

  const resetChart = (chartId: string) => {
    window.ApexCharts.exec(chartId, 'resetSeries');
  };

  return (
    <Styled.Content data-testid="metrics-deploy">
      <Styled.Card width="531px" height="79px">
        <Styled.FilterForm
          onSubmit={handleSubmit(onSubmit)}
          data-testid="metrics-filter"
        >
          <Styled.SingleSelect
            label="Select a timestamp"
            name="period"
            options={periodFilterItems}
            control={control}
            defaultValue={periodFilterItems[0]}
          />
          <CircleFilter control={control} setValue={setValue} />
          <Styled.Button
            type="submit"
            size="EXTRA_SMALL"
            isLoading={loading}
            data-testid="metrics-deploy-apply"
          >
            <Text.h5 weight="bold" align="center" color="light">
              Apply
            </Text.h5>
          </Styled.Button>
        </Styled.FilterForm>
      </Styled.Card>

      <Styled.Card width="1220px" height="521px" data-testid="apexchart-deploy">
        <ChartMenu onReset={() => resetChart('chartDeploy')} />
        <Styled.MixedChart
          options={deployOptions}
          series={deploySeries}
          width={1180}
          height={495}
        />
      </Styled.Card>
      <ReleasesHistoryComponent filter={filter} />
    </Styled.Content>
  );
};

export default Deploys;
