import React, { useState, useEffect } from 'react';
import api from '../../services/api';

function KPICard({ title, value, unit, testId }) {
  return (
    <div className="bg-white rounded-lg shadow p-6" data-testid={testId}>
      <p className="text-sm font-medium text-gray-500">{title}</p>
      <p className="mt-2 text-3xl font-bold text-gray-900">{value} <span className="text-lg font-normal text-gray-400">{unit}</span></p>
    </div>
  );
}

export default function KPIDashboardPage() {
  const [kpi, setKpi] = useState(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    api.get('/reports/kpi').then(r => setKpi(r.data.data)).finally(() => setLoading(false));
  }, []);

  if (loading) return <div className="text-center py-10 text-gray-500">Loading KPIs...</div>;

  return (
    <div>
      <h1 className="text-2xl font-bold text-gray-900 mb-6">KPI Dashboard</h1>
      <div className="grid grid-cols-1 md:grid-cols-3 gap-6 mb-8">
        <KPICard title="Conversion Rate" value={kpi ? `${(kpi.conversionRate * 100).toFixed(1)}%` : '—'} testId="kpi-conversion" />
        <KPICard title="Avg Review Cycle Time" value={kpi ? kpi.reviewCycleTime.toFixed(1) : '—'} unit="hrs" testId="kpi-cycle-time" />
        <KPICard title="Quota Utilization" value={kpi ? `${(kpi.quotaUtilization * 100).toFixed(1)}%` : '—'} testId="kpi-quota" />
      </div>
      <div className="bg-white rounded-lg shadow p-6">
        <h2 className="text-lg font-semibold mb-4">Export Preview (Sensitive Fields Masked)</h2>
        <p className="text-sm text-gray-500">Sensitive fields (demographics, exam scores, application details, transfer preferences) are masked or excluded for non-admin roles in CSV exports. Use the Export page to generate a scoped report.</p>
      </div>
    </div>
  );
}
