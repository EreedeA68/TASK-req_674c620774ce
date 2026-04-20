import React from 'react';
import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';
import { AuthProvider } from './context/AuthContext';
import RoleGuard from './components/RoleGuard';
import Navbar from './components/Navbar';
import LoginPage from './pages/auth/LoginPage';
import MFASetupPage from './pages/auth/MFASetupPage';
import CandidateListPage from './pages/candidates/CandidateListPage';
import CandidateDetailPage from './pages/candidates/CandidateDetailPage';
import CandidateFormPage from './pages/candidates/CandidateFormPage';
import DocumentUploadPage from './pages/candidates/DocumentUploadPage';
import ListingListPage from './pages/lostfound/ListingListPage';
import ListingDetailPage from './pages/lostfound/ListingDetailPage';
import ListingFormPage from './pages/lostfound/ListingFormPage';
import PartListPage from './pages/parts/PartListPage';
import PartDetailPage from './pages/parts/PartDetailPage';
import PartFormPage from './pages/parts/PartFormPage';
import PartVersionComparePage from './pages/parts/PartVersionComparePage';
import BulkImportPage from './pages/parts/BulkImportPage';
import KPIDashboardPage from './pages/reports/KPIDashboardPage';
import AuditLogPage from './pages/reports/AuditLogPage';
import AdminPage from './pages/admin/AdminPage';

function PrivateLayout({ children }) {
  return (
    <div className="min-h-screen bg-gray-50">
      <Navbar />
      <main className="max-w-7xl mx-auto px-4 py-6">{children}</main>
    </div>
  );
}

export default function App() {
  return (
    <AuthProvider>
      <BrowserRouter>
        <Routes>
          <Route path="/login" element={<LoginPage />} />
          <Route path="/unauthorized" element={<div className="flex items-center justify-center min-h-screen"><div className="text-center"><h1 className="text-2xl font-bold text-red-600">Unauthorized</h1><p className="mt-2 text-gray-600">You do not have permission to view this page.</p></div></div>} />

          <Route path="/mfa-setup" element={<PrivateLayout><RoleGuard><MFASetupPage /></RoleGuard></PrivateLayout>} />

          <Route path="/candidates" element={<PrivateLayout><RoleGuard roles={['ADMIN','INTAKE_SPECIALIST','REVIEWER','AUDITOR']}><CandidateListPage /></RoleGuard></PrivateLayout>} />
          <Route path="/candidates/new" element={<PrivateLayout><RoleGuard roles={['ADMIN','INTAKE_SPECIALIST']}><CandidateFormPage /></RoleGuard></PrivateLayout>} />
          <Route path="/candidates/:id" element={<PrivateLayout><RoleGuard roles={['ADMIN','INTAKE_SPECIALIST','REVIEWER','AUDITOR']}><CandidateDetailPage /></RoleGuard></PrivateLayout>} />
          <Route path="/candidates/:id/edit" element={<PrivateLayout><RoleGuard roles={['ADMIN','INTAKE_SPECIALIST']}><CandidateFormPage /></RoleGuard></PrivateLayout>} />
          <Route path="/candidates/:id/documents" element={<PrivateLayout><RoleGuard roles={['ADMIN','INTAKE_SPECIALIST']}><DocumentUploadPage /></RoleGuard></PrivateLayout>} />

          <Route path="/listings" element={<PrivateLayout><RoleGuard><ListingListPage /></RoleGuard></PrivateLayout>} />
          <Route path="/listings/new" element={<PrivateLayout><RoleGuard roles={['ADMIN','INVENTORY_CLERK']}><ListingFormPage /></RoleGuard></PrivateLayout>} />
          <Route path="/listings/:id" element={<PrivateLayout><RoleGuard><ListingDetailPage /></RoleGuard></PrivateLayout>} />
          <Route path="/listings/:id/edit" element={<PrivateLayout><RoleGuard roles={['ADMIN','INVENTORY_CLERK']}><ListingFormPage /></RoleGuard></PrivateLayout>} />

          <Route path="/parts" element={<PrivateLayout><RoleGuard><PartListPage /></RoleGuard></PrivateLayout>} />
          <Route path="/parts/new" element={<PrivateLayout><RoleGuard roles={['ADMIN','INVENTORY_CLERK']}><PartFormPage /></RoleGuard></PrivateLayout>} />
          <Route path="/parts/import" element={<PrivateLayout><RoleGuard roles={['ADMIN','INVENTORY_CLERK']}><BulkImportPage /></RoleGuard></PrivateLayout>} />
          <Route path="/parts/:id" element={<PrivateLayout><RoleGuard><PartDetailPage /></RoleGuard></PrivateLayout>} />
          <Route path="/parts/:id/edit" element={<PrivateLayout><RoleGuard roles={['ADMIN','INVENTORY_CLERK']}><PartFormPage /></RoleGuard></PrivateLayout>} />
          <Route path="/parts/:id/compare/:v1/:v2" element={<PrivateLayout><RoleGuard><PartVersionComparePage /></RoleGuard></PrivateLayout>} />

          <Route path="/reports/kpi" element={<PrivateLayout><RoleGuard roles={['ADMIN','AUDITOR']}><KPIDashboardPage /></RoleGuard></PrivateLayout>} />
          <Route path="/audit-logs" element={<PrivateLayout><RoleGuard roles={['ADMIN','AUDITOR']}><AuditLogPage /></RoleGuard></PrivateLayout>} />
          <Route path="/admin" element={<PrivateLayout><RoleGuard roles={['ADMIN']}><AdminPage /></RoleGuard></PrivateLayout>} />

          <Route path="/" element={<Navigate to="/candidates" replace />} />
          <Route path="*" element={<Navigate to="/login" replace />} />
        </Routes>
      </BrowserRouter>
    </AuthProvider>
  );
}
