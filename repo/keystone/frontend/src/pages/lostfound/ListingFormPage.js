import React, { useState, useEffect } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import api from '../../services/api';

const CATEGORIES = ['Accessories', 'Bags', 'Clothing', 'Documents', 'Electronics', 'Jewelry', 'Keys', 'Other', 'Pets', 'Wallet'];

const US_LOCATION_RE = /^[A-Za-z\s.\-]+,\s*[A-Z]{2}$/;
function isValidUSLocation(val) {
  return val && US_LOCATION_RE.test(val.trim());
}

export default function ListingFormPage() {
  const { id } = useParams();
  const navigate = useNavigate();
  const isEdit = Boolean(id);
  const [form, setForm] = useState({ title: '', category: '', locationDescription: '', timeWindowStart: '', timeWindowEnd: '' });
  const [errors, setErrors] = useState({});
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    if (isEdit) {
      api.get(`/listings/${id}`).then(r => {
        const l = r.data.data;
        setForm({ title: l.title, category: l.category, locationDescription: l.locationDescription,
          timeWindowStart: l.timeWindowStart?.slice(0,16) || '', timeWindowEnd: l.timeWindowEnd?.slice(0,16) || '' });
      });
    }
  }, [id, isEdit]);

  const set = k => e => setForm(f => ({ ...f, [k]: e.target.value }));

  const validate = () => {
    const e = {};
    if (!form.title.trim()) e.title = 'Title is required';
    if (!form.category) e.category = 'Category is required';
    if (!isValidUSLocation(form.locationDescription)) e.locationDescription = 'Enter a valid US location (e.g. "Chicago, IL")';
    if (form.timeWindowEnd && form.timeWindowStart && new Date(form.timeWindowEnd) <= new Date(form.timeWindowStart)) e.timeWindowEnd = 'End time must be after start time';
    setErrors(e);
    return Object.keys(e).length === 0;
  };

  const handleSubmit = async (e) => {
    e.preventDefault();
    if (!validate()) return;
    setLoading(true);
    try {
      const payload = {
        ...form,
        timeWindowStart: form.timeWindowStart ? new Date(form.timeWindowStart).toISOString() : null,
        timeWindowEnd:   form.timeWindowEnd   ? new Date(form.timeWindowEnd).toISOString()   : null,
      };
      if (isEdit) { await api.put(`/listings/${id}`, payload); navigate(`/listings/${id}`); }
      else { const r = await api.post('/listings', payload); navigate(`/listings/${r.data.data.id}`); }
    } catch (err) {
      setErrors({ submit: err.response?.data?.errorMessage || 'Save failed' });
    } finally { setLoading(false); }
  };

  return (
    <div className="max-w-xl mx-auto">
      <h1 className="text-2xl font-bold mb-6">{isEdit ? 'Edit Listing' : 'New Listing'}</h1>
      <form onSubmit={handleSubmit} className="bg-white rounded-lg shadow p-6 space-y-4">
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">Title</label>
          <input type="text" value={form.title} onChange={set('title')} className="w-full border rounded-md px-3 py-2 text-sm" data-testid="title-input" />
          {errors.title && <p className="text-red-600 text-xs mt-1" data-testid="title-error">{errors.title}</p>}
        </div>
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">Category</label>
          <select value={form.category} onChange={set('category')} className="w-full border rounded-md px-3 py-2 text-sm" data-testid="category-select">
            <option value="">Select category...</option>
            {CATEGORIES.map(c => <option key={c} value={c}>{c}</option>)}
          </select>
          {errors.category && <p className="text-red-600 text-xs mt-1" data-testid="category-error">{errors.category}</p>}
        </div>
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">Location (US)</label>
          <input type="text" value={form.locationDescription} onChange={set('locationDescription')} placeholder="e.g. Chicago, IL" className="w-full border rounded-md px-3 py-2 text-sm" data-testid="location-input" />
          {errors.locationDescription && <p className="text-red-600 text-xs mt-1" data-testid="location-error">{errors.locationDescription}</p>}
        </div>
        <div className="grid grid-cols-2 gap-4">
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Time Window Start</label>
            <input type="datetime-local" value={form.timeWindowStart} onChange={set('timeWindowStart')} className="w-full border rounded-md px-3 py-2 text-sm" data-testid="start-time" />
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Time Window End</label>
            <input type="datetime-local" value={form.timeWindowEnd} onChange={set('timeWindowEnd')} className="w-full border rounded-md px-3 py-2 text-sm" data-testid="end-time" />
            {errors.timeWindowEnd && <p className="text-red-600 text-xs mt-1" data-testid="time-error">{errors.timeWindowEnd}</p>}
          </div>
        </div>
        {errors.submit && <p className="text-red-600 text-sm">{errors.submit}</p>}
        <div className="flex gap-3">
          <button type="submit" disabled={loading} className="px-6 py-2 bg-blue-600 text-white text-sm rounded-md disabled:opacity-50">{loading ? 'Saving...' : 'Save'}</button>
          <button type="button" onClick={() => navigate(-1)} className="px-6 py-2 border text-sm rounded-md">Cancel</button>
        </div>
      </form>
    </div>
  );
}
