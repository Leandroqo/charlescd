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

package io.charlescd.moove.metrics.connector

import io.charlescd.moove.domain.MetricConfiguration
import io.charlescd.moove.metrics.connector.prometheus.PrometheusService
import org.springframework.context.ApplicationContext
import spock.lang.Specification

class MetricServiceImplUnitTest extends Specification {

    def context = Mock(ApplicationContext)
    def factoryService = new MetricServiceFactoryImpl(context)

    def 'should return PrometheusService when Provider type is Prometheus'() {
        when:
        def metricService = factoryService.getConnector(MetricConfiguration.ProviderEnum.PROMETHEUS)
        then:
        1 * context.getBean("prometheus") >> Mock(PrometheusService)
        0 * _

        metricService instanceof PrometheusService
    }

}
