import React, { useState, useEffect } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import api from '../../services/api';

export default function CandidateFormPage() {
  const { id } = useParams();
  const navigate = useNavigate();
  const isEdit = Boolean(id);
  const [form, setForm] = useState({
    firstName: '', lastName: '', dob: '', ssn: '', address: '',
    examScore: '', examDate: '', applicationDate: '', position: '',
    transferFrom: '', transferTo: ''
  });
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');

  useEffect(() => {
    if (isEdit) {
      api.get(`/candidates/${id}`).then(r => {
        const c = r.data.data;
        const demo = c.demographics || {};
        const exam = c.examScores || {};
        const app = c.applicationDetails || {};
        const transfer = c.transferPreferences || {};
        setForm({
          firstName: demo.firstName || '', lastName: demo.lastName || '',
          dob: demo.dob || '', ssn: demo.ssn || '', address: demo.address || '',
          examScore: exam.score || '', examDate: exam.date || '',
          applicationDate: app.applicationDate || '', position: app.position || '',
          transferFrom: transfer.from || '', transferTo: transfer.to || ''
        });
      });
    }
  }, [id, isEdit]);

  const set = (k) => (e) => setForm(f => ({ ...f, [k]: e.target.value }));

  const computeCompleteness = (f) => {
    const required = ['firstName','lastName','dob','examScore','applicationDate','position'];
    return required.every(k => f[k]) ? 'complete' : 'incomplete';
  };

  const handleSubmit = async (e) => {
    e.preventDefault();
    setError(''); setLoading(true);
    const payload = {
      demographics: { firstName: form.firstName, lastName: form.lastName, dob: form.dob, ssn: form.ssn, address: form.address },
      examScores: { score: form.examScore, date: form.examDate },
      applicationDetails: { applicationDate: form.applicationDate, position: form.position },
      transferPreferences: { from: form.transferFrom, to: form.transferTo },
      completenessStatus: computeCompleteness(form)
    };
    try {
      if (isEdit) {
        await api.put(`/candidates/${id}`, payload);
        navigate(`/candidates/${id}`);
      } else {
        const r = await api.post('/candidates', payload);
        navigate(`/candidates/${r.data.data.id}`);
      }
    } catch (err) {
      setError(err.response?.data?.errorMessage || 'Save failed');
    } finally {
      setLoading(false);
    }
  };

  const Field = ({ label, name, type = 'text' }) => (
    <div>
      <label className="block text-sm font-medium text-gray-700 mb-1">{label}</label>
      <input type={type} value={form[name]} onChange={set(name)}
        className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
        data-testid={`field-${name}`} />
    </div>
  );

  return (
    <div className="max-w-2xl mx-auto">
      <h1 className="text-2xl font-bold mb-6">{isEdit ? 'Edit Candidate' : 'New Candidate'}</h1>
      <form onSubmit={handleSubmit} className="bg-white rounded-lg shadow p-6 space-y-6">
        <div>
          <h2 className="text-lg font-semibold mb-3">Demographics</h2>
          <div className="grid grid-cols-2 gap-4">
            <Field label="First Name" name="firstName" />
            <Field label="Last Name" name="lastName" />
            <Field label="Date of Birth" name="dob" type="date" />
            <Field label="SSN (masked)" name="ssn" />
            <div className="col-span-2"><Field label="Address" name="address" /></div>
          </div>
        </div>
        <div>
          <h2 className="text-lg font-semibold mb-3">Exam Scores</h2>
          <div className="grid grid-cols-2 gap-4">
            <Field label="Score" name="examScore" />
            <Field label="Exam Date" name="examDate" type="date" />
          </div>
        </div>
        <div>
          <h2 className="text-lg font-semibold mb-3">Application Details</h2>
          <div className="grid grid-cols-2 gap-4">
            <Field label="Application Date" name="applicationDate" type="date" />
            <Field label="Position" name="position" />
          </div>
        </div>
        <div>
          <h2 className="text-lg font-semibold mb-3">Transfer Preferences</h2>
          <div className="grid grid-cols-2 gap-4">
            <Field label="Transfer From" name="transferFrom" />
            <Field label="Transfer To" name="transferTo" />
          </div>
        </div>
        {error && <p className="text-red-600 text-sm">{error}</p>}
        <div className="flex gap-3">
          <button type="submit" disabled={loading}
            className="px-6 py-2 bg-blue-600 text-white text-sm rounded-md hover:bg-blue-700 disabled:opacity-50">
            {loading ? 'Saving...' : 'Save'}
          </button>
          <button type="button" onClick={() => navigate(-1)} className="px-6 py-2 border text-sm rounded-md">Cancel</button>
        </div>
      </form>
    </div>
  );
}
