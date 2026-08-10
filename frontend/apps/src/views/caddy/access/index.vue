<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue';
import { useQueryClient } from '@tanstack/vue-query';

import { Page } from '@vben/common-ui';

import {
  Alert,
  Button,
  Card,
  Form,
  FormItem,
  Input,
  Radio,
  RadioGroup,
  Select,
  Space,
  Spin,
  Switch,
  Tag,
  message,
} from 'antdv-next';

import { getIpRegionConfigApi, updateIpRegionConfigApi } from '#/api/caddy/ip-region';
import { withListDetailErrorMode } from '#/api/list-detail';
import { invalidateListDetailQueries } from '#/api/list-detail-mutation';
import { qk } from '#/api/query-keys';
import { useListDetailQuery } from '#/composables/use-list-detail-query';

defineOptions({ name: 'CaddyAccess' });

type ChinaScope = 'all' | 'custom';

interface ChinaRegionRule {
  city: string;
  province: string;
}

const queryClient = useQueryClient();
const saving = ref(false);
const enabled = ref(true);
const allowList = ref<string[]>(['中国']);
const selectedCountries = ref<string[]>(['中国']);
const chinaScope = ref<ChinaScope>('all');
const chinaRules = ref<ChinaRegionRule[]>([]);
const seeded = ref(false);

const editingRule = reactive({
  city: '',
  province: '',
});

const countryOptions = [
  { label: '中国', value: '中国' },
  { label: '美国', value: '美国' },
  { label: '日本', value: '日本' },
  { label: '韩国', value: '韩国' },
  { label: '新加坡', value: '新加坡' },
  { label: '德国', value: '德国' },
  { label: '英国', value: '英国' },
  { label: '法国', value: '法国' },
  { label: '加拿大', value: '加拿大' },
  { label: '澳大利亚', value: '澳大利亚' },
  { label: '印度', value: '印度' },
  { label: '巴西', value: '巴西' },
  { label: '俄罗斯', value: '俄罗斯' },
  { label: '荷兰', value: '荷兰' },
  { label: '泰国', value: '泰国' },
  { label: '越南', value: '越南' },
  { label: '马来西亚', value: '马来西亚' },
  { label: '印度尼西亚', value: '印度尼西亚' },
  { label: '菲律宾', value: '菲律宾' },
  { label: '土耳其', value: '土耳其' },
];

const provinceOptions = [
  '北京市',
  '天津市',
  '河北省',
  '山西省',
  '内蒙古自治区',
  '辽宁省',
  '吉林省',
  '黑龙江省',
  '上海市',
  '江苏省',
  '浙江省',
  '安徽省',
  '福建省',
  '江西省',
  '山东省',
  '河南省',
  '湖北省',
  '湖南省',
  '广东省',
  '广西壮族自治区',
  '海南省',
  '重庆市',
  '四川省',
  '贵州省',
  '云南省',
  '西藏自治区',
  '陕西省',
  '甘肃省',
  '青海省',
  '宁夏回族自治区',
  '新疆维吾尔自治区',
  '香港特别行政区',
  '澳门特别行政区',
  '台湾省',
].map((item) => ({ label: item, value: item }));

const cityOptionsByProvince: Record<string, string[]> = {
  北京市: ['北京市'],
  上海市: ['上海市'],
  天津市: ['天津市'],
  重庆市: ['重庆市'],
  广东省: ['广州市', '深圳市', '珠海市', '佛山市', '东莞市', '中山市', '惠州市', '汕头市', '湛江市'],
  江苏省: ['南京市', '苏州市', '无锡市', '常州市', '南通市', '徐州市', '扬州市'],
  浙江省: ['杭州市', '宁波市', '温州市', '绍兴市', '嘉兴市', '金华市', '台州市'],
  四川省: ['成都市', '绵阳市', '德阳市', '宜宾市', '南充市', '乐山市', '泸州市'],
  山东省: ['济南市', '青岛市', '烟台市', '潍坊市', '临沂市', '济宁市'],
  河南省: ['郑州市', '洛阳市', '开封市', '新乡市', '南阳市'],
  湖北省: ['武汉市', '宜昌市', '襄阳市', '荆州市'],
  湖南省: ['长沙市', '株洲市', '湘潭市', '衡阳市', '岳阳市'],
  福建省: ['福州市', '厦门市', '泉州市', '漳州市', '莆田市'],
  安徽省: ['合肥市', '芜湖市', '蚌埠市', '阜阳市'],
  陕西省: ['西安市', '咸阳市', '宝鸡市', '榆林市'],
  辽宁省: ['沈阳市', '大连市', '鞍山市', '锦州市'],
  河北省: ['石家庄市', '唐山市', '保定市', '邯郸市', '廊坊市'],
  山西省: ['太原市', '大同市', '临汾市', '运城市'],
  江西省: ['南昌市', '赣州市', '九江市', '上饶市'],
  广西壮族自治区: ['南宁市', '柳州市', '桂林市', '北海市'],
  云南省: ['昆明市', '曲靖市', '大理白族自治州', '丽江市'],
  贵州省: ['贵阳市', '遵义市', '六盘水市'],
  海南省: ['海口市', '三亚市'],
  黑龙江省: ['哈尔滨市', '齐齐哈尔市', '大庆市'],
  吉林省: ['长春市', '吉林市'],
  内蒙古自治区: ['呼和浩特市', '包头市', '鄂尔多斯市'],
  甘肃省: ['兰州市', '天水市'],
  青海省: ['西宁市'],
  宁夏回族自治区: ['银川市'],
  新疆维吾尔自治区: ['乌鲁木齐市', '克拉玛依市'],
  西藏自治区: ['拉萨市'],
  香港特别行政区: ['香港特别行政区'],
  澳门特别行政区: ['澳门特别行政区'],
  台湾省: ['台北市', '高雄市', '台中市', '台南市'],
};

const chinaSelected = computed(() => selectedCountries.value.includes('中国'));
const editingCityOptions = computed(() =>
  (cityOptionsByProvince[editingRule.province] ?? []).map((item) => ({ label: item, value: item })),
);
const effectiveAllowList = computed(() => buildAllowList());

function normalizeRuleValue(value: string) {
  return value
    .split('/')
    .map((item) => item.trim())
    .filter(Boolean)
    .join('/');
}

function parseAllowList(values: string[]) {
  const countries: string[] = [];
  const rules: ChinaRegionRule[] = [];
  let hasChinaAll = false;

  for (const value of values) {
    const normalized = normalizeRuleValue(value);
    if (!normalized) continue;
    const parts = normalized.split('/');
    const country = parts[0] ?? '';
    const province = parts[1] ?? '';
    const city = parts[2] ?? '';
    if (!country) continue;
    if (country === '中国') {
      if (!province) {
        hasChinaAll = true;
        continue;
      }
      rules.push({ province, city });
      continue;
    }
    countries.push(country);
  }

  selectedCountries.value = Array.from(
    new Set(hasChinaAll || rules.length > 0 ? ['中国', ...countries] : countries),
  );
  chinaScope.value = hasChinaAll || rules.length === 0 ? 'all' : 'custom';
  chinaRules.value = dedupeChinaRules(rules);
}

function dedupeChinaRules(rules: ChinaRegionRule[]) {
  const seen = new Set<string>();
  return rules.filter((rule) => {
    const key = `${rule.province}/${rule.city}`;
    if (!rule.province || seen.has(key)) return false;
    seen.add(key);
    return true;
  });
}

function buildChinaRuleValue(rule: ChinaRegionRule) {
  return rule.city ? `中国/${rule.province}/${rule.city}` : `中国/${rule.province}`;
}

function buildAllowList() {
  const next = selectedCountries.value.filter((item) => item !== '中国');
  if (chinaSelected.value) {
    if (chinaScope.value === 'all' || chinaRules.value.length === 0) {
      next.unshift('中国');
    } else {
      next.unshift(...chinaRules.value.map(buildChinaRuleValue));
    }
  }
  return Array.from(new Set(next.map(normalizeRuleValue).filter(Boolean)));
}

const {
  data: remoteConfig,
  loading,
  errorMessage,
  refetch,
} = useListDetailQuery({
  queryKey: qk.caddy.ipRegion(),
  queryFn: () => getIpRegionConfigApi(withListDetailErrorMode()),
  errorFallback: '加载访问控制配置失败',
});

watch(
  remoteConfig,
  (data) => {
    if (!data || seeded.value) return;
    enabled.value = data.enabled;
    allowList.value = data.allowList?.length ? [...data.allowList] : ['中国'];
    parseAllowList(allowList.value);
    seeded.value = true;
  },
  { immediate: true },
);

async function handleSave() {
  saving.value = true;
  try {
    const nextAllowList = effectiveAllowList.value.length > 0 ? effectiveAllowList.value : ['中国'];
    await updateIpRegionConfigApi({
      enabled: enabled.value,
      allowList: nextAllowList,
    });
    allowList.value = nextAllowList;
    parseAllowList(nextAllowList);
    message.success('保存成功');
    seeded.value = false;
    await invalidateListDetailQueries(queryClient, qk.caddy.ipRegion());
    await refetch();
  } catch {
    message.error('保存失败');
  } finally {
    saving.value = false;
  }
}

function handleAddChinaRule() {
  if (!editingRule.province) {
    message.warning('请先选择省份');
    return;
  }
  const nextRule = {
    province: editingRule.province,
    city: editingRule.city.trim(),
  };
  chinaRules.value = dedupeChinaRules([...chinaRules.value, nextRule]);
  editingRule.city = '';
}

function handleRemoveChinaRule(index: number) {
  chinaRules.value.splice(index, 1);
}

watch(
  () => editingRule.province,
  () => {
    editingRule.city = '';
  },
);

watch(chinaSelected, (selected) => {
  if (!selected) {
    chinaRules.value = [];
  }
});
</script>

<template>
  <Page>
    <div class="access-page">
      <Card title="访问控制" variant="borderless" class="access-shell">
        <template #extra>
          <Tag :color="enabled ? 'green' : 'default'">
            {{ enabled ? '已启用' : '已禁用' }}
          </Tag>
        </template>

        <Spin :spinning="loading">
          <Alert
            v-if="errorMessage"
            class="mb-4"
            type="error"
            show-icon
            :message="errorMessage"
          />
          <Form layout="vertical" class="access-form">
            <FormItem label="启用 IP 区域限制">
              <Switch v-model:checked="enabled" />
            </FormItem>

            <FormItem label="允许访问的国家或地区">
              <Select
                v-model:value="selectedCountries"
                :options="countryOptions"
                :disabled="!enabled"
                mode="multiple"
                show-search
                placeholder="选择允许访问的国家或地区"
              />
            </FormItem>

            <FormItem v-if="chinaSelected" label="中国访问范围">
              <RadioGroup v-model:value="chinaScope" :disabled="!enabled">
                <Radio value="all">全国</Radio>
                <Radio value="custom">按省/市限制</Radio>
              </RadioGroup>
            </FormItem>

            <div v-if="chinaSelected && chinaScope === 'custom'" class="china-region-panel">
              <div class="china-rule-editor">
                <Select
                  v-model:value="editingRule.province"
                  :disabled="!enabled"
                  :options="provinceOptions"
                  class="region-control"
                  placeholder="选择省份"
                  show-search
                />
                <Select
                  v-model:value="editingRule.city"
                  :disabled="!enabled || !editingRule.province"
                  :options="editingCityOptions"
                  allow-clear
                  class="region-control"
                  placeholder="选择城市，可留空表示全省"
                  show-search
                />
                <Input
                  v-model:value="editingRule.city"
                  :disabled="!enabled || !editingRule.province"
                  class="region-control"
                  placeholder="或手动输入城市"
                />
                <Button :disabled="!enabled" class="region-add-button" @click="handleAddChinaRule">
                  添加
                </Button>
              </div>

              <div class="china-rule-list">
                <Tag
                  v-for="(rule, index) in chinaRules"
                  :key="`${rule.province}-${rule.city || 'all'}`"
                  closable
                  @close.prevent="handleRemoveChinaRule(index)"
                >
                  {{ rule.city ? `${rule.province} / ${rule.city}` : `${rule.province} / 全省` }}
                </Tag>
                <span v-if="chinaRules.length === 0" class="empty-tip">
                  还没有添加省市规则，保存时将按全国放行处理。
                </span>
              </div>
            </div>

            <FormItem label="当前生效范围">
              <Space wrap>
                <Tag v-for="item in effectiveAllowList" :key="item" color="blue">
                  {{ item.replaceAll('/', ' / ') }}
                </Tag>
              </Space>
            </FormItem>

            <FormItem label="说明">
              <Alert type="info" :show-icon="true">
                <template #message>
                  配置后，仅允许列表中的地区 IP 访问系统。选择“中国/省份”会放行整个省，选择“中国/省份/城市”只放行对应城市。修改后立即生效，无需重启服务。
                </template>
              </Alert>
            </FormItem>

            <FormItem>
              <Button type="primary" :loading="saving" :disabled="loading" @click="handleSave">
                保存
              </Button>
            </FormItem>
          </Form>
        </Spin>
      </Card>
    </div>
  </Page>
</template>

<style scoped>
.access-page {
  padding: 16px;
}

.access-shell {
  min-height: calc(100vh - 140px);
}

.access-form {
  max-width: 760px;
}

.china-region-panel {
  padding: 12px;
  margin-bottom: 24px;
  background: hsl(var(--muted));
  border: 1px solid hsl(var(--border));
  border-radius: 6px;
}

.china-rule-editor {
  display: grid;
  grid-template-columns: minmax(180px, 1fr) minmax(220px, 1fr) minmax(160px, 0.8fr) 80px;
  gap: 12px;
  align-items: center;
}

.region-control {
  width: 100%;
  min-width: 0;
}

/* 多选选中项：深色下避免浅底发灰/发白 */
.access-form :deep(.ant-select-selection-item) {
  color: hsl(var(--foreground));
  background: hsl(var(--accent));
  border-color: hsl(var(--border));
}

.access-form :deep(.ant-select-selection-item-remove) {
  color: hsl(var(--muted-foreground));
}

.china-rule-list :deep(.ant-tag) {
  color: hsl(var(--foreground));
  background: hsl(var(--accent));
  border-color: hsl(var(--border));
}

.region-add-button {
  width: 100%;
}

.china-rule-list {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-top: 12px;
}

.empty-tip {
  color: hsl(var(--muted-foreground));
  font-size: 13px;
}

@media (max-width: 900px) {
  .china-rule-editor {
    grid-template-columns: 1fr 1fr;
  }
}

@media (max-width: 640px) {
  .china-rule-editor {
    grid-template-columns: 1fr;
  }
}
</style>
