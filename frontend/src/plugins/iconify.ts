import { addIcon, addAPIProvider } from '@iconify/vue';

// MDI
import MonitorDashboard from '@iconify/icons-mdi/monitor-dashboard';
import FormatHorizontalAlignLeft from '@iconify/icons-mdi/format-horizontal-align-left';
import FormatHorizontalAlignRight from '@iconify/icons-mdi/format-horizontal-align-right';
import PinOffOutline from '@iconify/icons-mdi/pin-off-outline';
import PinOutline from '@iconify/icons-mdi/pin-outline';

// Carbon
import CloudMonitoring from '@iconify/icons-carbon/cloud-monitoring';
import Settings from '@iconify/icons-carbon/settings';
import Catalog from '@iconify/icons-carbon/catalog';
import UserRole from '@iconify/icons-carbon/user-role';
import CloudServiceManagement from '@iconify/icons-carbon/cloud-service-management';
import WarningFilled from '@iconify/icons-carbon/warning-filled';
import Unknown from '@iconify/icons-carbon/unknown';
import Http from '@iconify/icons-carbon/http';
import View from '@iconify/icons-carbon/view';
import User from '@iconify/icons-carbon/user';
import NetworkPublic from '@iconify/icons-carbon/network-public';
import Security from '@iconify/icons-carbon/security';
import WarningAlt from '@iconify/icons-carbon/warning-alt';
import DocumentDownload from '@iconify/icons-carbon/document-download';
import Add from '@iconify/icons-carbon/add';
import Edit from '@iconify/icons-carbon/edit';
import TrashCan from '@iconify/icons-carbon/trash-can';

// IC
import RoundManageAccounts from '@iconify/icons-ic/round-manage-accounts';
import RoundSearch from '@iconify/icons-ic/round-search';
import RoundRefresh from '@iconify/icons-ic/round-refresh';

// Ant Design
import CloseOutlined from '@iconify/icons-ant-design/close-outlined';
import ColumnWidthOutlined from '@iconify/icons-ant-design/column-width-outlined';
import LineOutlined from '@iconify/icons-ant-design/line-outlined';
import BarChartOutlined from '@iconify/icons-ant-design/bar-chart-outlined';
import MoneyCollectOutlined from '@iconify/icons-ant-design/money-collect-outlined';
import TrademarkCircleOutlined from '@iconify/icons-ant-design/trademark-circle-outlined';

// PH
import SignOut from '@iconify/icons-ph/sign-out';
import UserCircle from '@iconify/icons-ph/user-circle';

// Heroicons
import Language from '@iconify/icons-heroicons/language';

// Line MD
import MenuFoldLeft from '@iconify/icons-line-md/menu-fold-left';
import MenuFoldRight from '@iconify/icons-line-md/menu-fold-right';

// Majesticons
import ColorSwatchLine from '@iconify/icons-majesticons/color-swatch-line';

// Material Symbols
import Sunny from '@iconify/icons-material-symbols/sunny';
import NightlightRounded from '@iconify/icons-material-symbols/nightlight-rounded';
import HdrAuto from '@iconify/icons-material-symbols/hdr-auto';

export function setupIconifyOffline() {
  const { VITE_ICONIFY_URL } = import.meta.env;


  if (VITE_ICONIFY_URL) {
    addAPIProvider('', { resources: [VITE_ICONIFY_URL] });
  }

  // MDI
  addIcon('mdi:monitor-dashboard', MonitorDashboard);
  addIcon('mdi:format-horizontal-align-left', FormatHorizontalAlignLeft);
  addIcon('mdi:format-horizontal-align-right', FormatHorizontalAlignRight);
  addIcon('mdi:pin-off-outline', PinOffOutline);
  addIcon('mdi:pin-outline', PinOutline);

  // Carbon
  addIcon('carbon:cloud-monitoring', CloudMonitoring);
  addIcon('carbon:settings', Settings);
  addIcon('carbon:catalog', Catalog);
  addIcon('carbon:user-role', UserRole);
  addIcon('carbon:cloud-service-management', CloudServiceManagement);
  addIcon('carbon:warning-filled', WarningFilled);
  addIcon('carbon:unknown', Unknown);
  addIcon('carbon:http', Http);
  addIcon('carbon:view', View);
  addIcon('carbon:user', User);
  addIcon('carbon:network-public', NetworkPublic);
  addIcon('carbon:security', Security);
  addIcon('carbon:warning-alt', WarningAlt);
  addIcon('carbon:document-download', DocumentDownload);
  addIcon('carbon:add', Add);
  addIcon('carbon:edit', Edit);
  addIcon('carbon:trash-can', TrashCan);

  // IC
  addIcon('ic:round-manage-accounts', RoundManageAccounts);
  addIcon('ic:round-search', RoundSearch);
  addIcon('ic:round-refresh', RoundRefresh);

  // Ant Design
  addIcon('ant-design:close-outlined', CloseOutlined);
  addIcon('ant-design:column-width-outlined', ColumnWidthOutlined);
  addIcon('ant-design:line-outlined', LineOutlined);
  addIcon('ant-design:bar-chart-outlined', BarChartOutlined);
  addIcon('ant-design:money-collect-outlined', MoneyCollectOutlined);
  addIcon('ant-design:trademark-circle-outlined', TrademarkCircleOutlined);

  // PH
  addIcon('ph:sign-out', SignOut);
  addIcon('ph:user-circle', UserCircle);

  // Heroicons
  addIcon('heroicons:language', Language);

  // Line MD
  addIcon('line-md:menu-fold-left', MenuFoldLeft);
  addIcon('line-md:menu-fold-right', MenuFoldRight);

  // Majesticons
  addIcon('majesticons:color-swatch-line', ColorSwatchLine);

  // Material Symbols
  addIcon('material-symbols:sunny', Sunny);
  addIcon('material-symbols:nightlight-rounded', NightlightRounded);
  addIcon('material-symbols:hdr-auto', HdrAuto);
}
