import React, { useState, useEffect } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import api from '../../services/api';

export default function PartFormPage() {
  const { id } = useParams();
  const navigate = useNavigate();
  const isEdit = Boolean(id);
  const [form, setForm] = useState({
    partNumber: '', name: '', description: '', changeSummary: '',
    fitmentMake: '', fitmentModel: '', fitmentYear: '', fitmentEngine: '', fitmentTransmission: '',
    oemNumber: '', altNumber: ''
  });
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');

  useEffect(() => {
    if (isEdit) {
      api.get(`/parts/${id}`).then(r => {
        const p = r.data.data;
        setForm(f => ({ ...f, partNumber: p.partNumber, name: p.name, description: p.description || '' }));
      });
    }
  }, [id, isEdit]);

  const set = k => e => setForm(f => ({ ...f, [k]: e.target.value }));

  const handleSubmit = async (e) => {
    e.preventDefault();
    setError(''); setLoading(true);
    const payload = {
      partNumber: form.partNumber, name: form.name, description: form.description,
      changeSummary: form.changeSummary,
      fitment: { make: form.fitmentMake, model: form.fitmentModel, year: form.fitmentYear, engine: form.fitmentEngine, transmission: form.fitmentTransmission },
      oemMappings: form.oemNumber ? [{ oem: form.oemNumber, alt: form.altNumber }] : [],
      attributes: {}
    };
    try {
      if (isEdit) { await api.put(`/parts/${id}`, payload); navigate(`/parts/${id}`); }
      else { const r = await api.post('/parts', payload); navigate(`/parts/${r.data.data.id}`); }
    } catch (err) { setError(err.response?.data?.errorMessage || 'Save failed'); }
    finally { setLoading(false); }
  };

  const F = ({ label, name, type='text' }) => (
    <div>
      <label className="block text-sm font-medium text-gray-700 mb-1">{label}</label>
      <input type={type} value={form[name]} onChange={set(name)} className="w-full border rounded-md px-3 py-2 text-sm" data-testid={`field-${name}`} />
    </div>
  );

  return (
    <div className="max-w-2xl mx-auto">
      <h1 className="text-2xl font-bold mb-6">{isEdit ? 'Edit Part (New Version)' : 'New Part'}</h1>
      <form onSubmit={handleSubmit} className="bg-white rounded-lg shadow p-6 space-y-6">
        <div className="grid grid-cols-2 gap-4">
          {!isEdit && <F label="Part Number" name="partNumber" />}
          <F label="Name" name="name" />
          <div className="col-span-2"><F label="Description" name="description" /></div>
          {isEdit && <div className="col-span-2"><F label="Change Summary" name="changeSummary" /></div>}
        </div>
        <div>
          <h2 className="text-lg font-semibold mb-3">Fitment</h2>
          <div className="grid grid-cols-2 gap-4">
            <F label="Make" name="fitmentMake" />
            <F label="Model" name="fitmentModel" />
            <F label="Year" name="fitmentYear" type="number" />
            <F label="Engine" name="fitmentEngine" />
            <F label="Transmission" name="fitmentTransmission" />
          </div>
        </div>
        <div>
          <h2 className="text-lg font-semibold mb-3">OEM Mappings</h2>
          <div className="grid grid-cols-2 gap-4">
            <F label="OEM Number" name="oemNumber" />
            <F label="Alternative Number" name="altNumber" />
          </div>
        </div>
        {error && <p className="text-red-600 text-sm">{error}</p>}
        <div className="flex gap-3">
          <button type="submit" disabled={loading} className="px-6 py-2 bg-blue-600 text-white text-sm rounded-md disabled:opacity-50">{loading ? 'Saving...' : 'Save'}</button>
          <button type="button" onClick={() => navigate(-1)} className="px-6 py-2 border text-sm rounded-md">Cancel</button>
        </div>
      </form>
    </div>
  );
}
